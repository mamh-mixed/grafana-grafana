//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by deepcopy-gen. DO NOT EDIT.

package definition

import (
	v0alpha1 "github.com/grafana/grafana/pkg/apis/common/v0alpha1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QueryTypeDefinition) DeepCopyInto(out *QueryTypeDefinition) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QueryTypeDefinition.
func (in *QueryTypeDefinition) DeepCopy() *QueryTypeDefinition {
	if in == nil {
		return nil
	}
	out := new(QueryTypeDefinition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *QueryTypeDefinition) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QueryTypeDefinitionList) DeepCopyInto(out *QueryTypeDefinitionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]QueryTypeDefinition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QueryTypeDefinitionList.
func (in *QueryTypeDefinitionList) DeepCopy() *QueryTypeDefinitionList {
	if in == nil {
		return nil
	}
	out := new(QueryTypeDefinitionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *QueryTypeDefinitionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QueryTypeSpec) DeepCopyInto(out *QueryTypeSpec) {
	*out = *in
	if in.Versions != nil {
		in, out := &in.Versions, &out.Versions
		*out = make([]QueryTypeVersion, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QueryTypeSpec.
func (in *QueryTypeSpec) DeepCopy() *QueryTypeSpec {
	if in == nil {
		return nil
	}
	out := new(QueryTypeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QueryTypeVersion) DeepCopyInto(out *QueryTypeVersion) {
	*out = *in
	in.Schema.DeepCopyInto(&out.Schema)
	if in.Examples != nil {
		in, out := &in.Examples, &out.Examples
		*out = make([]v0alpha1.Unstructured, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Changelog != nil {
		in, out := &in.Changelog, &out.Changelog
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QueryTypeVersion.
func (in *QueryTypeVersion) DeepCopy() *QueryTypeVersion {
	if in == nil {
		return nil
	}
	out := new(QueryTypeVersion)
	in.DeepCopyInto(out)
	return out
}
