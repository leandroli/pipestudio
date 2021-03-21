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

// PipelineRunSpec defines the desired state of PipelineRun
type PipelineRunSpec struct {
	PipelineRef PipelineRef               `json:"pipelineRef"`
	Resources   []PipelineResourceBinding `json:"resources,omitempty"`
	// Params is a list of parameter names and values.
	Params         []Param `json:"params,omitempty"`
	ServiceAccount string  `json:"serviceAccount,omitempty"`
}

// PipelineRunStatus defines the observed state of PipelineRun
type PipelineRunStatus struct {
	TaskRuns map[string]TaskRunStatus `json:"taskRuns,omitempty"`
}

// PipelineRef defines the Pipeline to refer to
type PipelineRef struct {
	// Name of the referent
	Name string `json:"name"`
}

// PipelineResourceBinding is used to bind actual PipelineResource to the name
// of PipelineResource declared by Pipeline
type PipelineResourceBinding struct {
	// Name is the name of PipelineResource decleared in Pipeline
	Name string `json:"name"`
	// ResourceRef defines the PipelineResource to refer to
	ResourceRef PipelineResourceRef `json:"resourceRef"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PipelineRun is the Schema for the pipelineruns API
type PipelineRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineRunSpec   `json:"spec,omitempty"`
	Status PipelineRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineRunList contains a list of PipelineRun
type PipelineRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineRun{}, &PipelineRunList{})
}
