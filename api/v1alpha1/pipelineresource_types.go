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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
// PipelineResourceType represents the type of endpoint the pipelineResource is, now there is
// only one type git
type PipelineResourceType string

const (
	// PipelineResourceTypeGit indicates that this source is a GitHub repo.
	PipelineResourceTypeGit PipelineResourceType = "git"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Param declares a value to use for the Param called Name.
type Param struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PipelineResourceSpec defines the desired state of PipelineResource
type PipelineResourceSpec struct {
	Type   PipelineResourceType `json:"type"`
	Params []Param              `json:"params"`
}

// PipelineResourceRef can be used to refer to a specific instance of a Resource
type PipelineResourceRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// PipelineResourceStatus defines the observed state of PipelineResource
type PipelineResourceStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PipelineResource is the Schema for the pipelineresources API
type PipelineResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineResourceSpec   `json:"spec,omitempty"`
	Status PipelineResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineResourceList contains a list of PipelineResource
type PipelineResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineResource{}, &PipelineResourceList{})
}
