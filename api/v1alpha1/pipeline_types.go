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

type PipelineSpec struct {
	Params    []PipelineParam            `json:"params,omitempty"`
	Tasks     []PipelineTask             `json:"tasks"`
	Resources []PipelineDeclaredResource `json:"resources,omitempty"`
}

// PipelineStatus defines the observed state of Pipeline
type PipelineStatus struct {
}

// PipelineParam defines the parameter needed by Pipeline
type PipelineParam struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
}

// PipelineDeclaredResource describes names which can be used to refer to PipelineResources
// and types of them
type PipelineDeclaredResource struct {
	// Name is not correspond with actual name of PipelineResource, it is used by the Pipeline
	// to refer to PipelineResource
	Name string               `json:"name"`
	Type PipelineResourceType `json:"type"`
}

// PipelineTask defines a task in Pipeline
type PipelineTask struct {
	Name    string               `json:"name"`
	TaskRef TaskRef              `json:"taskRef"`
	Inputs  *PipelineTaskInputs  `json:"inputs,omitempty"`
	Outputs *PipelineTaskOutputs `json:"outputs,omitempty"`
}

type PipelineTaskInputs struct {
	Resources []PipelineTaskInputResource `json:"resources"`
	Params    []Param                     `json:"params,omitempty"`
}

type PipelineTaskOutputs struct {
	Resources []PipelineTaskOutputResource `json:"resources"`
}

// PipelineTaskInputResources map the DeclaredPipelineResources of Pipeline to the resources
// that required by tasks
type PipelineTaskInputResource struct {
	// Name is the name of the PipelineResource as declared by the Task.
	Name string `json:"name"`
	// Resource is the name of the DeclaredPipelineResource to use.
	Resource string `json:"resource"`
	From     string `json:"from,omitempty"`
}

// PipelineTaskOutputResources map the DeclaredPipelineResources of Pipeline to the resources
// that required by tasks
type PipelineTaskOutputResource struct {
	// Name is the name of the PipelineResource as declared by the Task.
	Name string `json:"name"`
	// Resource is the name of the DeclaredPipelineResource to use.
	Resource string `json:"resource"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Pipeline is the Schema for the pipelines API
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineSpec   `json:"spec,omitempty"`
	Status PipelineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineList contains a list of Pipeline
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pipeline{}, &PipelineList{})
}
