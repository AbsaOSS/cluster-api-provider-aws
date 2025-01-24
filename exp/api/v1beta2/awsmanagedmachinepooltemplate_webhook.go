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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager will setup the webhooks for the AWSManagedMachinePool.
func (rt *AWSManagedMachinePoolTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(rt).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-awsmanagedmachinepooltemplate,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=awsmanagedmachinepooltemplates,versions=v1beta2,name=validation.awsmanagedmachinepooltemplate.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1beta2-awsmanagedmachinepooltemplate,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=awsmanagedmachinepooltemplates,versions=v1beta2,name=default.awsmanagedmachinepooltemplate.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var _ webhook.Defaulter = &AWSManagedMachinePool{}
var _ webhook.Validator = &AWSManagedMachinePool{}

// ValidateCreate will do any extra validation when creating a AWSManagedMachinePoolTemplate.
func (rt *AWSManagedMachinePoolTemplate) ValidateCreate() (admission.Warnings, error) {
	mmpLog.Info("AWSManagedMachinePoolTemplate validate create", "managed-machine-pool", klog.KObj(rt))

	var allErrs field.ErrorList

	if errs := validateScaling(rt.Spec.Template.Spec); errs != nil || len(errs) == 0 {
		allErrs = append(allErrs, errs...)
	}

	if errs := validateRemoteAccess(rt.Spec.Template.Spec); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if errs := validateNodegroupUpdateConfig(rt.Spec.Template.Spec); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if errs := validateLaunchTemplate(rt.Spec.Template.Spec); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	allErrs = append(allErrs, rt.Spec.Template.Spec.AdditionalTags.Validate()...)

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		rt.GroupVersionKind().GroupKind(),
		rt.Name,
		allErrs,
	)
}

// ValidateUpdate will do any extra validation when creating a AWSManagedMachinePoolTemplate.
func (rt *AWSManagedMachinePoolTemplate) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	mmpLog.Info("AWSManagedMachinePoolTemplate validate update", "managed-machine-pool", klog.KObj(rt))
	oldPoolTemplate, ok := old.(*AWSManagedMachinePoolTemplate)
	if !ok {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("AWSManagedMachinePoolTemplate").GroupKind(), rt.Name, field.ErrorList{
			field.InternalError(nil, errors.New("failed to convert old AWSManagedMachinePool to object")),
		})
	}

	var allErrs field.ErrorList
	allErrs = append(allErrs, validateAMPImmutable(rt.Spec.Template.Spec, oldPoolTemplate.Spec.Template.Spec)...)
	allErrs = append(allErrs, rt.Spec.Template.Spec.AdditionalTags.Validate()...)

	if errs := validateScaling(rt.Spec.Template.Spec); errs != nil || len(errs) == 0 {
		allErrs = append(allErrs, errs...)
	}
	if errs := validateNodegroupUpdateConfig(rt.Spec.Template.Spec); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}
	if errs := validateLaunchTemplate(rt.Spec.Template.Spec); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(
		rt.GroupVersionKind().GroupKind(),
		rt.Name,
		allErrs,
	)

}

// ValidateDelete will do any extra validation when creating a AWSManagedMachinePoolTemplate.
func (rt *AWSManagedMachinePoolTemplate) ValidateDelete() (admission.Warnings, error) {
	mmpLog.Info("AWSManagedMachinePoolTemplate validate delete", "managed-machine-pool", klog.KObj(rt))

	return nil, nil
}

// Default will set default values for the AWSManagedMachinePool.
func (rt *AWSManagedMachinePoolTemplate) Default() {
	if rt.Spec.Template.Spec.UpdateConfig == nil {
		rt.Spec.Template.Spec.UpdateConfig = defaultManagedMachinePoolUpdateConfig()
	}
}
