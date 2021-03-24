// +build !ignore_autogenerated

/*
Copyright 2021.

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

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScaledActionRunner) DeepCopyInto(out *ScaledActionRunner) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScaledActionRunner.
func (in *ScaledActionRunner) DeepCopy() *ScaledActionRunner {
	if in == nil {
		return nil
	}
	out := new(ScaledActionRunner)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ScaledActionRunner) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScaledActionRunnerList) DeepCopyInto(out *ScaledActionRunnerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ScaledActionRunner, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScaledActionRunnerList.
func (in *ScaledActionRunnerList) DeepCopy() *ScaledActionRunnerList {
	if in == nil {
		return nil
	}
	out := new(ScaledActionRunnerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ScaledActionRunnerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScaledActionRunnerSpec) DeepCopyInto(out *ScaledActionRunnerSpec) {
	*out = *in
	if in.RunnerSecrets != nil {
		in, out := &in.RunnerSecrets, &out.RunnerSecrets
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.WorkVolumeSize != nil {
		in, out := &in.WorkVolumeSize, &out.WorkVolumeSize
		x := (*in).DeepCopy()
		*out = &x
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScaledActionRunnerSpec.
func (in *ScaledActionRunnerSpec) DeepCopy() *ScaledActionRunnerSpec {
	if in == nil {
		return nil
	}
	out := new(ScaledActionRunnerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScaledActionRunnerStatus) DeepCopyInto(out *ScaledActionRunnerStatus) {
	*out = *in
	if in.ReferencedSecrets != nil {
		in, out := &in.ReferencedSecrets, &out.ReferencedSecrets
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScaledActionRunnerStatus.
func (in *ScaledActionRunnerStatus) DeepCopy() *ScaledActionRunnerStatus {
	if in == nil {
		return nil
	}
	out := new(ScaledActionRunnerStatus)
	in.DeepCopyInto(out)
	return out
}