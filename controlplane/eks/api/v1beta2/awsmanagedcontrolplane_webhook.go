/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	"fmt"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	infrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/eks"
)

const (
	minAddonVersion          = "v1.18.0"
	minKubeVersionForIPv6    = "v1.21.0"
	minVpcCniVersionForIPv6  = "1.10.2"
	maxClusterNameLength     = 100
	hostnameTypeResourceName = "resource-name"
)

// log is for logging in this package.
var mcpLog = ctrl.Log.WithName("awsmanagedcontrolplane-resource")

const (
	cidrSizeMax    = 65536
	cidrSizeMin    = 16
	vpcCniAddon    = "vpc-cni"
	kubeProxyAddon = "kube-proxy"
)

// SetupWebhookWithManager will setup the webhooks for the AWSManagedControlPlane.
func (r *AWSManagedControlPlane) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-controlplane-cluster-x-k8s-io-v1beta2-awsmanagedcontrolplane,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=controlplane.cluster.x-k8s.io,resources=awsmanagedcontrolplanes,versions=v1beta2,name=validation.awsmanagedcontrolplanes.controlplane.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-controlplane-cluster-x-k8s-io-v1beta2-awsmanagedcontrolplane,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=controlplane.cluster.x-k8s.io,resources=awsmanagedcontrolplanes,versions=v1beta2,name=default.awsmanagedcontrolplanes.controlplane.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var _ webhook.Defaulter = &AWSManagedControlPlane{}
var _ webhook.Validator = &AWSManagedControlPlane{}

func parseEKSVersion(raw string) (*version.Version, error) {
	v, err := version.ParseGeneric(raw)
	if err != nil {
		return nil, err
	}
	return version.MustParseGeneric(fmt.Sprintf("%d.%d", v.Major(), v.Minor())), nil
}

// ValidateCreate will do any extra validation when creating a AWSManagedControlPlane.
func (r *AWSManagedControlPlane) ValidateCreate() (admission.Warnings, error) {
	mcpLog.Info("AWSManagedControlPlane validate create", "control-plane", klog.KObj(r))

	var allErrs field.ErrorList

	if r.Spec.EKSClusterName == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec.eksClusterName"), "eksClusterName is required"))
	}

	// TODO: Add ipv6 validation things in these validations.
	allErrs = append(allErrs, validateEKSVersion(&r.Spec, nil)...)
	allErrs = append(allErrs, r.Spec.Bastion.Validate()...)
	allErrs = append(allErrs, validateIAMAuthConfig(r.Spec)...)
	allErrs = append(allErrs, validateSecondaryCIDR(r.Spec)...)
	allErrs = append(allErrs, validateEKSAddons(r.Spec)...)
	allErrs = append(allErrs, validateDisableVPCCNI(r.Spec)...)
	allErrs = append(allErrs, validateRestrictPrivateSubnets(r.Spec)...)
	allErrs = append(allErrs, validateKubeProxy(r.Spec)...)
	allErrs = append(allErrs, r.Spec.AdditionalTags.Validate()...)
	allErrs = append(allErrs, validateNetwork(r.Spec)...)
	allErrs = append(allErrs, validatePrivateDNSHostnameTypeOnLaunch(r.Spec)...)

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		r.GroupVersionKind().GroupKind(),
		r.Name,
		allErrs,
	)
}

// ValidateUpdate will do any extra validation when updating a AWSManagedControlPlane.
func (r *AWSManagedControlPlane) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	mcpLog.Info("AWSManagedControlPlane validate update", "control-plane", klog.KObj(r))
	oldAWSManagedControlplane, ok := old.(*AWSManagedControlPlane)
	if !ok {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("AWSManagedControlPlane").GroupKind(), r.Name, field.ErrorList{
			field.InternalError(nil, errors.New("failed to convert old AWSManagedControlPlane to object")),
		})
	}

	var allErrs field.ErrorList
	allErrs = append(allErrs, r.validateEKSClusterName()...)
	allErrs = append(allErrs, validateEKSClusterNameSame(r.Spec, oldAWSManagedControlplane.Spec)...)
	allErrs = append(allErrs, validateEKSVersion(&r.Spec, &oldAWSManagedControlplane.Spec)...)
	allErrs = append(allErrs, r.Spec.Bastion.Validate()...)
	allErrs = append(allErrs, validateAccessConfig(r.Spec, oldAWSManagedControlplane.Spec)...)
	allErrs = append(allErrs, validateIAMAuthConfig(r.Spec)...)
	allErrs = append(allErrs, validateSecondaryCIDR(r.Spec)...)
	allErrs = append(allErrs, validateEKSAddons(r.Spec)...)
	allErrs = append(allErrs, validateDisableVPCCNI(r.Spec)...)
	allErrs = append(allErrs, validateRestrictPrivateSubnets(r.Spec)...)
	allErrs = append(allErrs, validateKubeProxy(r.Spec)...)
	allErrs = append(allErrs, r.Spec.AdditionalTags.Validate()...)
	allErrs = append(allErrs, validatePrivateDNSHostnameTypeOnLaunch(r.Spec)...)

	if r.Spec.Region != oldAWSManagedControlplane.Spec.Region {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "region"), r.Spec.Region, "field is immutable"),
		)
	}

	// If encryptionConfig is already set, do not allow removal of it.
	if oldAWSManagedControlplane.Spec.EncryptionConfig != nil && r.Spec.EncryptionConfig == nil {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "encryptionConfig"), r.Spec.EncryptionConfig, "disabling EKS encryption is not allowed after it has been enabled"),
		)
	}

	// If encryptionConfig is already set, do not allow change in provider
	if r.Spec.EncryptionConfig != nil &&
		r.Spec.EncryptionConfig.Provider != nil &&
		oldAWSManagedControlplane.Spec.EncryptionConfig != nil &&
		oldAWSManagedControlplane.Spec.EncryptionConfig.Provider != nil &&
		*r.Spec.EncryptionConfig.Provider != *oldAWSManagedControlplane.Spec.EncryptionConfig.Provider {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "encryptionConfig", "provider"), r.Spec.EncryptionConfig.Provider, "changing EKS encryption is not allowed after it has been enabled"),
		)
	}

	// If a identityRef is already set, do not allow removal of it.
	if oldAWSManagedControlplane.Spec.IdentityRef != nil && r.Spec.IdentityRef == nil {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "identityRef"),
				r.Spec.IdentityRef, "field cannot be set to nil"),
		)
	}

	if oldAWSManagedControlplane.Spec.NetworkSpec.VPC.IsIPv6Enabled() != r.Spec.NetworkSpec.VPC.IsIPv6Enabled() {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "network", "vpc", "enableIPv6"), r.Spec.NetworkSpec.VPC.IsIPv6Enabled(), "changing IP family is not allowed after it has been set"))
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		r.GroupVersionKind().GroupKind(),
		r.Name,
		allErrs,
	)
}

// ValidateDelete allows you to add any extra validation when deleting.
func (r *AWSManagedControlPlane) ValidateDelete() (admission.Warnings, error) {
	mcpLog.Info("AWSManagedControlPlane validate delete", "control-plane", klog.KObj(r))

	return nil, nil
}

func (r *AWSManagedControlPlane) validateEKSClusterName() field.ErrorList {
	var allErrs field.ErrorList

	if r.Spec.EKSClusterName == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec.eksClusterName"), "eksClusterName is required"))
	}

	return allErrs
}

func validateEKSClusterNameSame(r, old AWSManagedControlPlaneSpec) field.ErrorList {
	var allErrs field.ErrorList
	if old.EKSClusterName != "" && r.EKSClusterName != old.EKSClusterName {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec.eksClusterName"), r.EKSClusterName, "eksClusterName is different to current cluster name"))
	}

	return allErrs
}

func validateEKSVersion(r, old *AWSManagedControlPlaneSpec) field.ErrorList {
	path := field.NewPath("spec.version")
	var allErrs field.ErrorList

	if r.Version == nil {
		return allErrs
	}

	v, err := parseEKSVersion(*r.Version)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(path, *r.Version, err.Error()))
	}

	if old != nil && old.Version != nil {
		oldV, err := parseEKSVersion(*old.Version)
		if err == nil && (v.Major() < oldV.Major() || v.Minor() < oldV.Minor()) {
			allErrs = append(allErrs, field.Invalid(path, *r.Version, "new version less than old version"))
		}
	}

	if r.NetworkSpec.VPC.IsIPv6Enabled() {
		minIPv6, _ := version.ParseSemantic(minKubeVersionForIPv6)
		if v.LessThan(minIPv6) {
			allErrs = append(allErrs, field.Invalid(path, *r.Version, fmt.Sprintf("IPv6 requires Kubernetes %s or greater", minKubeVersionForIPv6)))
		}
	}
	return allErrs
}

func validateEKSAddons(r AWSManagedControlPlaneSpec) field.ErrorList {
	var allErrs field.ErrorList

	if !r.NetworkSpec.VPC.IsIPv6Enabled() && (r.Addons == nil || len(*r.Addons) == 0) {
		return allErrs
	}

	if r.Version == nil {
		return allErrs
	}

	path := field.NewPath("spec.version")
	v, err := parseEKSVersion(*r.Version)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(path, *r.Version, err.Error()))
	}

	minVersion, _ := version.ParseSemantic(minAddonVersion)

	addonsPath := field.NewPath("spec.addons")

	if v.LessThan(minVersion) {
		message := fmt.Sprintf("addons requires Kubernetes %s or greater", minAddonVersion)
		allErrs = append(allErrs, field.Invalid(addonsPath, *r.Version, message))
	}

	// validations for IPv6:
	// - addons have to be defined in case IPv6 is enabled
	// - minimum version requirement for VPC-CNI using IPv6 ipFamily is 1.10.2
	if r.NetworkSpec.VPC.IsIPv6Enabled() {
		if r.Addons == nil || len(*r.Addons) == 0 {
			allErrs = append(allErrs, field.Invalid(addonsPath, "", "addons are required to be set explicitly if IPv6 is enabled"))
			return allErrs
		}

		for _, addon := range *r.Addons {
			if addon.Name == vpcCniAddon {
				v, err := version.ParseGeneric(addon.Version)
				if err != nil {
					allErrs = append(allErrs, field.Invalid(addonsPath, addon.Version, err.Error()))
					break
				}
				minCniVersion, _ := version.ParseSemantic(minVpcCniVersionForIPv6)
				if v.LessThan(minCniVersion) {
					allErrs = append(allErrs, field.Invalid(addonsPath, addon.Version, fmt.Sprintf("vpc-cni version must be above or equal to %s for IPv6", minVpcCniVersionForIPv6)))
					break
				}
			}
		}
	}

	return allErrs
}

func validateAccessConfig(r, old AWSManagedControlPlaneSpec) field.ErrorList {
	var allErrs field.ErrorList

	// If accessConfig is already set, do not allow removal of it.
	if old.AccessConfig != nil && r.AccessConfig == nil {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "accessConfig"), r.AccessConfig, "removing AccessConfig is not allowed after it has been enabled"),
		)
	}

	// AuthenticationMode is ratcheting - do not allow downgrades
	if old.AccessConfig != nil && old.AccessConfig.AuthenticationMode != r.AccessConfig.AuthenticationMode &&
		((old.AccessConfig.AuthenticationMode == EKSAuthenticationModeAPIAndConfigMap && r.AccessConfig.AuthenticationMode == EKSAuthenticationModeConfigMap) ||
			old.AccessConfig.AuthenticationMode == EKSAuthenticationModeAPI) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "accessConfig", "authenticationMode"), r.AccessConfig.AuthenticationMode, "downgrading authentication mode is not allowed after it has been enabled"),
		)
	}

	return allErrs
}

func validateIAMAuthConfig(r AWSManagedControlPlaneSpec) field.ErrorList {
	var allErrs field.ErrorList

	parentPath := field.NewPath("spec.iamAuthenticatorConfig")

	cfg := r.IAMAuthenticatorConfig
	if cfg == nil {
		return allErrs
	}

	for i, userMapping := range cfg.UserMappings {
		usersPathName := fmt.Sprintf("mapUsers[%d]", i)
		usersPath := parentPath.Child(usersPathName)
		errs := userMapping.Validate()
		for _, validErr := range errs {
			allErrs = append(allErrs, field.Invalid(usersPath, userMapping, validErr.Error()))
		}
	}

	for i, roleMapping := range cfg.RoleMappings {
		rolePathName := fmt.Sprintf("mapRoles[%d]", i)
		rolePath := parentPath.Child(rolePathName)
		errs := roleMapping.Validate()
		for _, validErr := range errs {
			allErrs = append(allErrs, field.Invalid(rolePath, roleMapping, validErr.Error()))
		}
	}

	return allErrs
}

func validateSecondaryCIDR(r AWSManagedControlPlaneSpec) field.ErrorList {
	var allErrs field.ErrorList
	if r.SecondaryCidrBlock != nil {
		cidrField := field.NewPath("spec", "secondaryCidrBlock")
		_, validRange1, _ := net.ParseCIDR("100.64.0.0/10")
		_, validRange2, _ := net.ParseCIDR("198.19.0.0/16")

		_, ipv4Net, err := net.ParseCIDR(*r.SecondaryCidrBlock)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(cidrField, *r.SecondaryCidrBlock, "must be valid CIDR range"))
			return allErrs
		}

		cidrSize := cidr.AddressCount(ipv4Net)
		if cidrSize > cidrSizeMax || cidrSize < cidrSizeMin {
			allErrs = append(allErrs, field.Invalid(cidrField, *r.SecondaryCidrBlock, "CIDR block sizes must be between a /16 netmask and /28 netmask"))
		}

		start, end := cidr.AddressRange(ipv4Net)
		if (!validRange1.Contains(start) || !validRange1.Contains(end)) && (!validRange2.Contains(start) || !validRange2.Contains(end)) {
			allErrs = append(allErrs, field.Invalid(cidrField, *r.SecondaryCidrBlock, "must be within the 100.64.0.0/10 or 198.19.0.0/16 range"))
		}
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func validateKubeProxy(r AWSManagedControlPlaneSpec) field.ErrorList {
	var allErrs field.ErrorList

	if r.KubeProxy.Disable {
		disableField := field.NewPath("spec", "kubeProxy", "disable")

		if r.Addons != nil {
			for _, addon := range *r.Addons {
				if addon.Name == kubeProxyAddon {
					allErrs = append(allErrs, field.Invalid(disableField, r.KubeProxy.Disable, "cannot disable kube-proxy if the kube-proxy addon is specified"))
					break
				}
			}
		}
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func validateDisableVPCCNI(r AWSManagedControlPlaneSpec) field.ErrorList {
	var allErrs field.ErrorList

	if r.VpcCni.Disable {
		disableField := field.NewPath("spec", "vpcCni", "disable")

		if r.Addons != nil {
			for _, addon := range *r.Addons {
				if addon.Name == vpcCniAddon {
					allErrs = append(allErrs, field.Invalid(disableField, r.VpcCni.Disable, "cannot disable vpc cni if the vpc-cni addon is specified"))
					break
				}
			}
		}
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func validateRestrictPrivateSubnets(r AWSManagedControlPlaneSpec) field.ErrorList {
	var allErrs field.ErrorList

	if r.RestrictPrivateSubnets && r.NetworkSpec.VPC.IsUnmanaged(r.EKSClusterName) {
		boolField := field.NewPath("spec", "restrictPrivateSubnets")
		if len(r.NetworkSpec.Subnets.FilterPrivate()) == 0 {
			allErrs = append(allErrs, field.Invalid(boolField, r.RestrictPrivateSubnets, "cannot enable private subnets restriction when no private subnets are specified"))
		}
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func validatePrivateDNSHostnameTypeOnLaunch(r AWSManagedControlPlaneSpec) field.ErrorList {
	var allErrs field.ErrorList

	if r.NetworkSpec.VPC.IsIPv6Enabled() && r.NetworkSpec.VPC.PrivateDNSHostnameTypeOnLaunch != nil && *r.NetworkSpec.VPC.PrivateDNSHostnameTypeOnLaunch != hostnameTypeResourceName {
		privateDNSHostnameTypeOnLaunch := field.NewPath("spec", "networkSpec", "vpc", "privateDNSHostnameTypeOnLaunch")
		allErrs = append(allErrs, field.Invalid(privateDNSHostnameTypeOnLaunch, r.NetworkSpec.VPC.PrivateDNSHostnameTypeOnLaunch, fmt.Sprintf("only %s HostnameType can be used in IPv6 mode", hostnameTypeResourceName)))
	}

	return allErrs
}

func validateNetwork(r AWSManagedControlPlaneSpec) field.ErrorList {
	var allErrs field.ErrorList

	// If only `AWSManagedControlPlane.spec.secondaryCidrBlock` is set, no additional checks are done to remain
	// backward-compatible. The `VPCSpec.SecondaryCidrBlocks` field was added later - if that list is not empty, we
	// require `AWSManagedControlPlane.spec.secondaryCidrBlock` to be listed in there as well. This may allow merging
	// the fields later on.
	podSecondaryCidrBlock := r.SecondaryCidrBlock
	secondaryCidrBlocks := r.NetworkSpec.VPC.SecondaryCidrBlocks
	secondaryCidrBlocksField := field.NewPath("spec", "network", "vpc", "secondaryCidrBlocks")
	if podSecondaryCidrBlock != nil && len(secondaryCidrBlocks) > 0 {
		found := false
		for _, cidrBlock := range secondaryCidrBlocks {
			if cidrBlock.IPv4CidrBlock == *podSecondaryCidrBlock {
				found = true
				break
			}
		}
		if !found {
			allErrs = append(allErrs, field.Invalid(secondaryCidrBlocksField, secondaryCidrBlocks, fmt.Sprintf("AWSManagedControlPlane.spec.secondaryCidrBlock %v must be listed in AWSManagedControlPlane.spec.network.vpc.secondaryCidrBlocks (required if both fields are filled)", *podSecondaryCidrBlock)))
		}
	}

	if podSecondaryCidrBlock != nil && r.NetworkSpec.VPC.CidrBlock != "" && r.NetworkSpec.VPC.CidrBlock == *podSecondaryCidrBlock {
		secondaryCidrBlockField := field.NewPath("spec", "vpc", "secondaryCidrBlock")
		allErrs = append(allErrs, field.Invalid(secondaryCidrBlockField, secondaryCidrBlocks, fmt.Sprintf("AWSManagedControlPlane.spec.secondaryCidrBlock %v must not be equal to the primary AWSManagedControlPlane.spec.network.vpc.cidrBlock", *podSecondaryCidrBlock)))
	}
	for _, cidrBlock := range secondaryCidrBlocks {
		if r.NetworkSpec.VPC.CidrBlock != "" && r.NetworkSpec.VPC.CidrBlock == cidrBlock.IPv4CidrBlock {
			allErrs = append(allErrs, field.Invalid(secondaryCidrBlocksField, secondaryCidrBlocks, fmt.Sprintf("AWSManagedControlPlane.spec.network.vpc.secondaryCidrBlocks must not contain the primary AWSManagedControlPlane.spec.network.vpc.cidrBlock %v", r.NetworkSpec.VPC.CidrBlock)))
		}
	}

	if r.NetworkSpec.VPC.IsIPv6Enabled() && r.NetworkSpec.VPC.IPv6.CidrBlock != "" && r.NetworkSpec.VPC.IPv6.PoolID == "" {
		poolField := field.NewPath("spec", "network", "vpc", "ipv6", "poolId")
		allErrs = append(allErrs, field.Invalid(poolField, r.NetworkSpec.VPC.IPv6.PoolID, "poolId cannot be empty if cidrBlock is set"))
	}

	if r.NetworkSpec.VPC.IsIPv6Enabled() && r.NetworkSpec.VPC.IPv6.PoolID != "" && r.NetworkSpec.VPC.IPv6.IPAMPool != nil {
		poolField := field.NewPath("spec", "network", "vpc", "ipv6", "poolId")
		allErrs = append(allErrs, field.Invalid(poolField, r.NetworkSpec.VPC.IPv6.PoolID, "poolId and ipamPool cannot be used together"))
	}

	if r.NetworkSpec.VPC.IsIPv6Enabled() && r.NetworkSpec.VPC.IPv6.CidrBlock != "" && r.NetworkSpec.VPC.IPv6.IPAMPool != nil {
		cidrBlockField := field.NewPath("spec", "network", "vpc", "ipv6", "cidrBlock")
		allErrs = append(allErrs, field.Invalid(cidrBlockField, r.NetworkSpec.VPC.IPv6.CidrBlock, "cidrBlock and ipamPool cannot be used together"))
	}

	if r.NetworkSpec.VPC.IsIPv6Enabled() && r.NetworkSpec.VPC.IPv6.IPAMPool != nil && r.NetworkSpec.VPC.IPv6.IPAMPool.ID == "" && r.NetworkSpec.VPC.IPv6.IPAMPool.Name == "" {
		ipamPoolField := field.NewPath("spec", "network", "vpc", "ipv6", "ipamPool")
		allErrs = append(allErrs, field.Invalid(ipamPoolField, r.NetworkSpec.VPC.IPv6.IPAMPool, "ipamPool must have either id or name"))
	}

	return allErrs
}

// Default will set default values for the AWSManagedControlPlane.
func (r *AWSManagedControlPlane) Default() {
	mcpLog.Info("AWSManagedControlPlane setting defaults", "control-plane", klog.KObj(r))

	if r.Spec.EKSClusterName == "" {
		mcpLog.Info("EKSClusterName is empty, generating name")
		name, err := eks.GenerateEKSName(r.Name, r.Namespace, maxClusterNameLength)
		if err != nil {
			mcpLog.Error(err, "failed to create EKS cluster name")
			return
		}

		mcpLog.Info("defaulting EKS cluster name", "cluster", klog.KRef(r.Namespace, name))
		r.Spec.EKSClusterName = name
	}

	if r.Spec.IdentityRef == nil {
		r.Spec.IdentityRef = &infrav1.AWSIdentityReference{
			Kind: infrav1.ControllerIdentityKind,
			Name: infrav1.AWSClusterControllerIdentityName,
		}
	}

	infrav1.SetDefaults_Bastion(&r.Spec.Bastion)
	infrav1.SetDefaults_NetworkSpec(&r.Spec.NetworkSpec)
}
