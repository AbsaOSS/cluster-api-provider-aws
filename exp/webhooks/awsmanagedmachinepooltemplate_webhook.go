/*
Copyright 2026 The Kubernetes Authors.

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

package webhooks

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	expinfrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/exp/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/eks"
)

// log is for logging in this package.
var mmptLog = ctrl.Log.WithName("AWSManagedMachinePoolTemplatetemplate-resource")

// AWSManagedMachinePoolTemplate implements a custom validation webhook for AWSManagedMachinePoolTemplate.
type AWSManagedMachinePoolTemplate struct{}

// SetupWebhookWithManager will setup the webhooks for the AWSManagedMachinePoolTemplate.
func (w *AWSManagedMachinePoolTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&expinfrav1.AWSManagedMachinePoolTemplate{}).
		WithValidator(w).
		WithDefaulter(w).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-AWSManagedMachinePoolTemplate,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=AWSManagedMachinePoolTemplates,versions=v1beta2,name=validation.AWSManagedMachinePoolTemplate.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1beta2-AWSManagedMachinePoolTemplate,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=AWSManagedMachinePoolTemplates,versions=v1beta2,name=default.AWSManagedMachinePoolTemplate.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var _ webhook.CustomDefaulter = &AWSManagedMachinePoolTemplate{}
var _ webhook.CustomValidator = &AWSManagedMachinePoolTemplate{}

func (w *AWSManagedMachinePoolTemplate) validateScaling(r *expinfrav1.AWSManagedMachinePoolTemplate) field.ErrorList {
	var allErrs field.ErrorList
	if r.Spec.Template.Spec.Scaling != nil { //nolint:nestif
		minField := field.NewPath("spec", "scaling", "minSize")
		maxField := field.NewPath("spec", "scaling", "maxSize")
		minSize := r.Spec.Template.Spec.Scaling.MinSize
		maxSize := r.Spec.Template.Spec.Scaling.MaxSize
		if minSize != nil {
			if *minSize < 0 {
				allErrs = append(allErrs, field.Invalid(minField, *minSize, "must be greater or equal zero"))
			}
			if maxSize != nil && *maxSize < *minSize {
				allErrs = append(allErrs, field.Invalid(maxField, *maxSize, fmt.Sprintf("must be greater than field %s", minField.String())))
			}
		}
		if maxSize != nil && *maxSize < 0 {
			allErrs = append(allErrs, field.Invalid(maxField, *maxSize, "must be greater than zero"))
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func (w *AWSManagedMachinePoolTemplate) validateNodegroupUpdateConfig(r *expinfrav1.AWSManagedMachinePoolTemplate) field.ErrorList {
	var allErrs field.ErrorList

	if r.Spec.Template.Spec.UpdateConfig != nil {
		nodegroupUpdateConfigField := field.NewPath("spec", "updateConfig")

		if r.Spec.Template.Spec.UpdateConfig.MaxUnavailable == nil && r.Spec.Template.Spec.UpdateConfig.MaxUnavailablePercentage == nil {
			allErrs = append(allErrs, field.Invalid(nodegroupUpdateConfigField, r.Spec.Template.Spec.UpdateConfig, "must specify one of maxUnavailable or maxUnavailablePercentage when using nodegroup updateconfig"))
		}

		if r.Spec.Template.Spec.UpdateConfig.MaxUnavailable != nil && r.Spec.Template.Spec.UpdateConfig.MaxUnavailablePercentage != nil {
			allErrs = append(allErrs, field.Invalid(nodegroupUpdateConfigField, r.Spec.Template.Spec.UpdateConfig, "cannot specify both maxUnavailable and maxUnavailablePercentage"))
		}
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func (w *AWSManagedMachinePoolTemplate) validateRemoteAccess(r *expinfrav1.AWSManagedMachinePoolTemplate) field.ErrorList {
	var allErrs field.ErrorList
	if r.Spec.Template.Spec.RemoteAccess == nil {
		return allErrs
	}
	remoteAccessPath := field.NewPath("spec", "remoteAccess")
	sourceSecurityGroups := r.Spec.Template.Spec.RemoteAccess.SourceSecurityGroups

	if public := r.Spec.Template.Spec.RemoteAccess.Public; public && len(sourceSecurityGroups) > 0 {
		allErrs = append(
			allErrs,
			field.Invalid(remoteAccessPath.Child("sourceSecurityGroups"), sourceSecurityGroups, "must be empty if public is set"),
		)
	}

	return allErrs
}

func (w *AWSManagedMachinePoolTemplate) validateLaunchTemplate(r *expinfrav1.AWSManagedMachinePoolTemplate) field.ErrorList {
	var allErrs field.ErrorList
	if r.Spec.Template.Spec.AWSLaunchTemplate == nil {
		return allErrs
	}

	if r.Spec.Template.Spec.InstanceType != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "InstanceType"), r.Spec.Template.Spec.InstanceType, "InstanceType cannot be specified when LaunchTemplate is specified"))
	}
	if r.Spec.Template.Spec.DiskSize != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "DiskSize"), r.Spec.Template.Spec.DiskSize, "DiskSize cannot be specified when LaunchTemplate is specified"))
	}

	if r.Spec.Template.Spec.AWSLaunchTemplate.IamInstanceProfile != "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "AWSLaunchTemplate", "IamInstanceProfile"), r.Spec.Template.Spec.AWSLaunchTemplate.IamInstanceProfile, "IAM instance profile in launch template is prohibited in EKS managed node group"))
	}

	return allErrs
}

func (w *AWSManagedMachinePoolTemplate) validateLifecycleHooks(r *expinfrav1.AWSManagedMachinePoolTemplate) field.ErrorList {
	return validateLifecycleHooks(r.Spec.Template.Spec.AWSLifecycleHooks)
}

// ValidateCreate will do any extra validation when creating a AWSManagedMachinePoolTemplate.
func (w *AWSManagedMachinePoolTemplate) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*expinfrav1.AWSManagedMachinePoolTemplate)
	if !ok {
		return nil, fmt.Errorf("expected an AWSManagedMachinePoolTemplate object but got %T", r)
	}

	mmptLog.Info("AWSManagedMachinePoolTemplate validate create", "managed-machine-pool", klog.KObj(r))

	var allErrs field.ErrorList

	if r.Spec.Template.Spec.EKSNodegroupName == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec.eksNodegroupName"), "eksNodegroupName is required"))
	}
	if errs := w.validateScaling(r); errs != nil || len(errs) == 0 {
		allErrs = append(allErrs, errs...)
	}
	if errs := w.validateRemoteAccess(r); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}
	if errs := w.validateNodegroupUpdateConfig(r); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}
	if errs := w.validateLaunchTemplate(r); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}
	if errs := w.validateLifecycleHooks(r); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	allErrs = append(allErrs, r.Spec.Template.Spec.AdditionalTags.Validate()...)

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		r.GroupVersionKind().GroupKind(),
		r.Name,
		allErrs,
	)
}

// ValidateUpdate will do any extra validation when updating a AWSManagedMachinePoolTemplate.
func (w *AWSManagedMachinePoolTemplate) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	r, ok := newObj.(*expinfrav1.AWSManagedMachinePoolTemplate)
	if !ok {
		return nil, fmt.Errorf("expected an AWSManagedMachinePoolTemplate object but got %T", r)
	}

	mmptLog.Info("AWSManagedMachinePoolTemplate validate update", "managed-machine-pool", klog.KObj(r))
	oldPool, ok := oldObj.(*expinfrav1.AWSManagedMachinePoolTemplate)
	if !ok {
		return nil, apierrors.NewInvalid(expinfrav1.GroupVersion.WithKind("AWSManagedMachinePoolTemplate").GroupKind(), r.Name, field.ErrorList{
			field.InternalError(nil, errors.New("failed to convert old AWSManagedMachinePoolTemplate to object")),
		})
	}

	var allErrs field.ErrorList
	allErrs = append(allErrs, w.validateImmutable(r, oldPool)...)
	allErrs = append(allErrs, r.Spec.Template.Spec.AdditionalTags.Validate()...)

	if errs := w.validateScaling(r); errs != nil || len(errs) == 0 {
		allErrs = append(allErrs, errs...)
	}
	if errs := w.validateNodegroupUpdateConfig(r); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}
	if errs := w.validateLaunchTemplate(r); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}
	if errs := w.validateLifecycleHooks(r); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
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
func (w *AWSManagedMachinePoolTemplate) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (w *AWSManagedMachinePoolTemplate) validateImmutable(r *expinfrav1.AWSManagedMachinePoolTemplate, old *expinfrav1.AWSManagedMachinePoolTemplate) field.ErrorList {
	var allErrs field.ErrorList

	appendErrorIfMutated := func(old, update interface{}, name string) {
		if !cmp.Equal(old, update) {
			allErrs = append(
				allErrs,
				field.Invalid(field.NewPath("spec", name), update, "field is immutable"),
			)
		}
	}
	appendErrorIfSetAndMutated := func(old, update interface{}, name string) {
		if !reflect.ValueOf(old).IsZero() && !cmp.Equal(old, update) {
			allErrs = append(
				allErrs,
				field.Invalid(field.NewPath("spec", name), update, "field is immutable"),
			)
		}
	}

	if old.Spec.Template.Spec.EKSNodegroupName != "" {
		appendErrorIfMutated(old.Spec.Template.Spec.EKSNodegroupName, r.Spec.Template.Spec.EKSNodegroupName, "eksNodegroupName")
	}
	appendErrorIfMutated(old.Spec.Template.Spec.SubnetIDs, r.Spec.Template.Spec.SubnetIDs, "subnetIDs")
	appendErrorIfSetAndMutated(old.Spec.Template.Spec.RoleName, r.Spec.Template.Spec.RoleName, "roleName")
	appendErrorIfMutated(old.Spec.Template.Spec.DiskSize, r.Spec.Template.Spec.DiskSize, "diskSize")
	appendErrorIfMutated(old.Spec.Template.Spec.AMIType, r.Spec.Template.Spec.AMIType, "amiType")
	appendErrorIfMutated(old.Spec.Template.Spec.RemoteAccess, r.Spec.Template.Spec.RemoteAccess, "remoteAccess")
	appendErrorIfSetAndMutated(old.Spec.Template.Spec.CapacityType, r.Spec.Template.Spec.CapacityType, "capacityType")
	appendErrorIfMutated(old.Spec.Template.Spec.AvailabilityZones, r.Spec.Template.Spec.AvailabilityZones, "availabilityZones")
	appendErrorIfMutated(old.Spec.Template.Spec.AvailabilityZoneSubnetType, r.Spec.Template.Spec.AvailabilityZoneSubnetType, "availabilityZoneSubnetType")
	if (old.Spec.Template.Spec.AWSLaunchTemplate != nil && r.Spec.Template.Spec.AWSLaunchTemplate == nil) ||
		(old.Spec.Template.Spec.AWSLaunchTemplate == nil && r.Spec.Template.Spec.AWSLaunchTemplate != nil) {
		allErrs = append(
			allErrs,
			field.Invalid(field.NewPath("spec", "AWSLaunchTemplate"), old.Spec.Template.Spec.AWSLaunchTemplate, "field is immutable"),
		)
	}
	if old.Spec.Template.Spec.AWSLaunchTemplate != nil && r.Spec.Template.Spec.AWSLaunchTemplate != nil {
		appendErrorIfMutated(old.Spec.Template.Spec.AWSLaunchTemplate.Name, r.Spec.Template.Spec.AWSLaunchTemplate.Name, "awsLaunchTemplate.name")
	}

	return allErrs
}

// Default will set default values for the AWSManagedMachinePoolTemplate.
func (w *AWSManagedMachinePoolTemplate) Default(_ context.Context, obj runtime.Object) error {
	r, ok := obj.(*expinfrav1.AWSManagedMachinePoolTemplate)
	if !ok {
		return fmt.Errorf("expected an AWSManagedMachinePoolTemplate object but got %T", r)
	}

	mmptLog.Info("AWSManagedMachinePoolTemplate setting defaults", "managed-machine-pool", klog.KObj(r))

	if r.Spec.Template.Spec.EKSNodegroupName == "" {
		mmptLog.Info("EKSNodegroupName is empty, generating name")
		name, err := eks.GenerateEKSName(r.Name, r.Namespace, maxNodegroupNameLength)
		if err != nil {
			mmptLog.Error(err, "failed to create EKS nodegroup name")
			return nil
		}

		mmptLog.Info("Generated EKSNodegroupName", "nodegroup", klog.KRef(r.Namespace, name))
		r.Spec.Template.Spec.EKSNodegroupName = name
	}

	if r.Spec.Template.Spec.UpdateConfig == nil {
		r.Spec.Template.Spec.UpdateConfig = &expinfrav1.UpdateConfig{
			MaxUnavailable: ptr.To[int](1),
		}
	}
	return nil
}
