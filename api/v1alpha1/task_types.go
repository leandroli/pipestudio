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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TaskSpec defines the desired state of Task
type TaskSpec struct {
	// Steps are the steps of the build
	Steps []corev1.Container `json:"steps,omitempty"`

	// Volumes is a collection of volumes that are available to mount into the
	// steps of the build.
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	Inputs *Inputs `json:"inputs,omitempty"`

	Outputs *Outputs `json:"outputs,omitempty"`
}

// TaskResource defines an input or output Resource declared as a requirement
// by a Task. The Name field will be used to refer to these Resources within
// the Task definition, and when provided as an Input, the Name will be the
// path to the volume mounted containing this Resource as an input (e.g.
// an input Resource named `workspace` will be mounted at `/workspace`).
type TaskResource struct {
	Name string               `json:"name"`
	Type PipelineResourceType `json:"type"`
	// TargetPath is the path in workspace directory where the task resource will be copied.
	TargetPath string `json:"targetPath,omitempty"`
}

// TaskParam defines arbitrary parameters needed by a task beyond typed inputs
// such as resources.
type TaskParam struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

// Inputs are the requirements that a task needs to run a Build.
type Inputs struct {
	Resources []TaskResource `json:"resources,omitempty"`
	Params    []TaskParam    `json:"params,omitempty"`
}

// Outputs allow a task to declare what data the Build/Task will be producing,
// i.e. results such as logs and artifacts such as images.
type Outputs struct {
	Resources []TaskResource `json:"resources,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Task is the Schema for the tasks API
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TaskSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// TaskList contains a list of Task
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Task{}, &TaskList{})
}
