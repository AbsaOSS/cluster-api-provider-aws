//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	apiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EKSConfig) DeepCopyInto(out *EKSConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EKSConfig.
func (in *EKSConfig) DeepCopy() *EKSConfig {
	if in == nil {
		return nil
	}
	out := new(EKSConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EKSConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EKSConfigList) DeepCopyInto(out *EKSConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EKSConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EKSConfigList.
func (in *EKSConfigList) DeepCopy() *EKSConfigList {
	if in == nil {
		return nil
	}
	out := new(EKSConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EKSConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EKSConfigSpec) DeepCopyInto(out *EKSConfigSpec) {
	*out = *in
	if in.KubeletExtraArgs != nil {
		in, out := &in.KubeletExtraArgs, &out.KubeletExtraArgs
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ContainerRuntime != nil {
		in, out := &in.ContainerRuntime, &out.ContainerRuntime
		*out = new(string)
		**out = **in
	}
	if in.DNSClusterIP != nil {
		in, out := &in.DNSClusterIP, &out.DNSClusterIP
		*out = new(string)
		**out = **in
	}
	if in.DockerConfigJSON != nil {
		in, out := &in.DockerConfigJSON, &out.DockerConfigJSON
		*out = new(string)
		**out = **in
	}
	if in.APIRetryAttempts != nil {
		in, out := &in.APIRetryAttempts, &out.APIRetryAttempts
		*out = new(int)
		**out = **in
	}
	if in.PauseContainer != nil {
		in, out := &in.PauseContainer, &out.PauseContainer
		*out = new(PauseContainer)
		**out = **in
	}
	if in.UseMaxPods != nil {
		in, out := &in.UseMaxPods, &out.UseMaxPods
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EKSConfigSpec.
func (in *EKSConfigSpec) DeepCopy() *EKSConfigSpec {
	if in == nil {
		return nil
	}
	out := new(EKSConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EKSConfigStatus) DeepCopyInto(out *EKSConfigStatus) {
	*out = *in
	if in.DataSecretName != nil {
		in, out := &in.DataSecretName, &out.DataSecretName
		*out = new(string)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(apiv1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EKSConfigStatus.
func (in *EKSConfigStatus) DeepCopy() *EKSConfigStatus {
	if in == nil {
		return nil
	}
	out := new(EKSConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EKSConfigTemplate) DeepCopyInto(out *EKSConfigTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EKSConfigTemplate.
func (in *EKSConfigTemplate) DeepCopy() *EKSConfigTemplate {
	if in == nil {
		return nil
	}
	out := new(EKSConfigTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EKSConfigTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EKSConfigTemplateList) DeepCopyInto(out *EKSConfigTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EKSConfigTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EKSConfigTemplateList.
func (in *EKSConfigTemplateList) DeepCopy() *EKSConfigTemplateList {
	if in == nil {
		return nil
	}
	out := new(EKSConfigTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EKSConfigTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EKSConfigTemplateResource) DeepCopyInto(out *EKSConfigTemplateResource) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EKSConfigTemplateResource.
func (in *EKSConfigTemplateResource) DeepCopy() *EKSConfigTemplateResource {
	if in == nil {
		return nil
	}
	out := new(EKSConfigTemplateResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EKSConfigTemplateSpec) DeepCopyInto(out *EKSConfigTemplateSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EKSConfigTemplateSpec.
func (in *EKSConfigTemplateSpec) DeepCopy() *EKSConfigTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(EKSConfigTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PauseContainer) DeepCopyInto(out *PauseContainer) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PauseContainer.
func (in *PauseContainer) DeepCopy() *PauseContainer {
	if in == nil {
		return nil
	}
	out := new(PauseContainer)
	in.DeepCopyInto(out)
	return out
}