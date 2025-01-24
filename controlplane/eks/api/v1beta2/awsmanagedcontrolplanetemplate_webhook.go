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
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	infrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager will setup the webhooks for the AWSManagedControlPlaneTemplate.
func (rt *AWSManagedControlPlaneTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(rt).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-controlplane-cluster-x-k8s-io-v1beta2-awsmanagedcontrolplanetemplate,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=controlplane.cluster.x-k8s.io,resources=awsmanagedcontrolplanetemplatess,versions=v1beta2,name=validation.awsmanagedcontrolplanetemplates.controlplane.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-controlplane-cluster-x-k8s-io-v1beta2-awsmanagedcontrolplanetemplate,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=controlplane.cluster.x-k8s.io,resources=awsmanagedcontrolplanetemplatess,versions=v1beta2,name=default.awsmanagedcontrolplanetemplates.controlplane.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var _ webhook.Defaulter = &AWSManagedControlPlaneTemplate{}
var _ webhook.Validator = &AWSManagedControlPlaneTemplate{}

func (rt *AWSManagedControlPlaneTemplate) ValidateCreate() (admission.Warnings, error) {
	mcpLog.Info("AWSManagedControlPlane validate create", "control-plane-template", klog.KObj(rt))
	var allErrs field.ErrorList
	allErrs = append(allErrs, validateEKSVersion(&rt.Spec.Template.Spec, nil)...)
	allErrs = append(allErrs, rt.Spec.Template.Spec.Bastion.Validate()...)
	allErrs = append(allErrs, validateIAMAuthConfig(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateSecondaryCIDR(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateEKSAddons(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateDisableVPCCNI(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateRestrictPrivateSubnets(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateKubeProxy(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, rt.Spec.Template.Spec.AdditionalTags.Validate()...)
	allErrs = append(allErrs, validateNetwork(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validatePrivateDNSHostnameTypeOnLaunch(rt.Spec.Template.Spec)...)

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		rt.GroupVersionKind().GroupKind(),
		rt.Name,
		allErrs,
	)
}
func (rt *AWSManagedControlPlaneTemplate) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	oldAWSManagedControlplaneTemplate, ok := old.(*AWSManagedControlPlaneTemplate)
	if !ok {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("AWSManagedControlPlaneTemplate").GroupKind(), rt.Name, field.ErrorList{
			field.InternalError(nil, errors.New("failed to convert old AWSManagedControlPlaneTemplate to object")),
		})
	}
	var allErrs field.ErrorList
	allErrs = append(allErrs, validateEKSClusterNameSame(rt.Spec.Template.Spec, oldAWSManagedControlplaneTemplate.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateEKSVersion(&rt.Spec.Template.Spec, &oldAWSManagedControlplaneTemplate.Spec.Template.Spec)...)
	allErrs = append(allErrs, rt.Spec.Template.Spec.Bastion.Validate()...)
	allErrs = append(allErrs, validateAccessConfig(rt.Spec.Template.Spec, oldAWSManagedControlplaneTemplate.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateIAMAuthConfig(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateSecondaryCIDR(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateEKSAddons(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateDisableVPCCNI(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateRestrictPrivateSubnets(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, validateKubeProxy(rt.Spec.Template.Spec)...)
	allErrs = append(allErrs, rt.Spec.Template.Spec.AdditionalTags.Validate()...)
	allErrs = append(allErrs, validatePrivateDNSHostnameTypeOnLaunch(rt.Spec.Template.Spec)...)

	if rt.Spec.Template.Spec.Region != oldAWSManagedControlplaneTemplate.Spec.Template.Spec.Region {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "region"), rt.Spec.Template.Spec.Region, "field is immutable"),
		)
	}

	// If encryptionConfig is already set, do not allow removal of it.
	if oldAWSManagedControlplaneTemplate.Spec.Template.Spec.EncryptionConfig != nil && rt.Spec.Template.Spec.EncryptionConfig == nil {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "encryptionConfig"), rt.Spec.Template.Spec.EncryptionConfig, "disabling EKS encryption is not allowed after it has been enabled"),
		)
	}

	// If encryptionConfig is already set, do not allow change in provider
	if rt.Spec.Template.Spec.EncryptionConfig != nil &&
		rt.Spec.Template.Spec.EncryptionConfig.Provider != nil &&
		oldAWSManagedControlplaneTemplate.Spec.Template.Spec.EncryptionConfig != nil &&
		oldAWSManagedControlplaneTemplate.Spec.Template.Spec.EncryptionConfig.Provider != nil &&
		*rt.Spec.Template.Spec.EncryptionConfig.Provider != *oldAWSManagedControlplaneTemplate.Spec.Template.Spec.EncryptionConfig.Provider {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "encryptionConfig", "provider"), rt.Spec.Template.Spec.EncryptionConfig.Provider, "changing EKS encryption is not allowed after it has been enabled"),
		)
	}

	// If a identityRef is already set, do not allow removal of it.
	if oldAWSManagedControlplaneTemplate.Spec.Template.Spec.IdentityRef != nil && rt.Spec.Template.Spec.IdentityRef == nil {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "identityRef"),
				rt.Spec.Template.Spec.IdentityRef, "field cannot be set to nil"),
		)
	}

	if oldAWSManagedControlplaneTemplate.Spec.Template.Spec.NetworkSpec.VPC.IsIPv6Enabled() != rt.Spec.Template.Spec.NetworkSpec.VPC.IsIPv6Enabled() {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "network", "vpc", "enableIPv6"), rt.Spec.Template.Spec.NetworkSpec.VPC.IsIPv6Enabled(), "changing IP family is not allowed after it has been set"))
	}

	return nil, apierrors.NewInvalid(
		rt.GroupVersionKind().GroupKind(),
		rt.Name,
		allErrs,
	)
}

func (rt *AWSManagedControlPlaneTemplate) ValidateDelete() (admission.Warnings, error) {
	mcpLog.Info("AWSManagedControlPlaneTemplate validate delete", "control-plane", klog.KObj(rt))
	return nil, nil
}

func (rt *AWSManagedControlPlaneTemplate) Default() {
	mcpLog.Info("AWSManagedControlPlaneTemplate setting defaults", "control-plane", klog.KObj(rt))

	if rt.Spec.Template.Spec.IdentityRef == nil {
		rt.Spec.Template.Spec.IdentityRef = &infrav1.AWSIdentityReference{
			Kind: infrav1.ControllerIdentityKind,
			Name: infrav1.AWSClusterControllerIdentityName,
		}
	}

	infrav1.SetDefaults_Bastion(&rt.Spec.Template.Spec.Bastion)
	infrav1.SetDefaults_NetworkSpec(&rt.Spec.Template.Spec.NetworkSpec)
}
