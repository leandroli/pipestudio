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

// TaskRunSpec defines the desired state of TaskRun
type TaskRunSpec struct {
	TaskRef        TaskRef         `json:"taskRef,omitempty"`
	Trigger        TaskTrigger     `json:"trigger,omitempty"`
	Inputs         *TaskRunInputs  `json:"inputs,omitempty"`
	Outputs        *TaskRunOutputs `json:"outputs,omitempty"`
	ServiceAccount string          `json:"serviceAccount,omitempty"`
}

// TaskRunInputs holds the input values that this task was invoked with.
type TaskRunInputs struct {
	Resources []TaskResourceBinding `json:"resources,omitempty"`
	Params    []Param               `json:"params,omitempty"`
}

// TaskRunOutputs holds the output values that this task was invoked with.
type TaskRunOutputs struct {
	Resources []TaskResourceBinding `json:"resources,omitempty"`
	Params    []Param               `json:"params,omitempty"`
}

// TaskResourceBinding points to the PipelineResource that
// will be used for the Task input or output called Name.
type TaskResourceBinding struct {
	Name string `json:"name"`
	// no more than one of the ResourceRef and ResourceSpec may be specified.
	ResourceRef PipelineResourceRef `json:"resourceRef"`
}

// TaskRef can be used to refer to a specific instance of a task.
type TaskRef struct {
	Name string `json:"name"`
}

// TaskTriggerType indicates the mechanism by which this TaskRun was created.
type TaskTriggerType string

const (
	// TaskTriggerTypeManual indicates that this TaskRun was invoked manually by a user.
	TaskTriggerTypeManual TaskTriggerType = "manual"

	// TaskTriggerTypePipelineRun indicates that this TaskRun was created by a controller
	// attempting to realize a PipelineRun. In this case the `name` will refer to the name
	// of the PipelineRun.
	TaskTriggerTypePipelineRun TaskTriggerType = "pipelineRun"
)

// TaskTrigger describes what triggered this Task to run. It could be triggered manually,
// or it may have been part of a PipelineRun in which case this ref would refer
// to the corresponding PipelineRun.
type TaskTrigger struct {
	Type TaskTriggerType `json:"type"`
	Name string          `json:"name,omitempty"`
}

// TaskRunStatus defines the observed state of TaskRun
type TaskRunStatus struct {
	// Steps describes the state of each build step container.
	Steps []corev1.ContainerStatus `json:"steps,omitempty"`

	// PodName is the name of the pod responsible for executing this task's steps.
	PodName string `json:"podName"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TaskRun is the Schema for the taskruns API
type TaskRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskRunSpec   `json:"spec,omitempty"`
	Status TaskRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TaskRunList contains a list of TaskRun
type TaskRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskRun `json:"items"`
}

// GetBuildPodRef for task
func (tr *TaskRun) GetBuildPodRef() corev1.ObjectReference {
	return corev1.ObjectReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Namespace:  tr.Namespace,
		Name:       tr.Name,
	}
}

// GetBuildPodMeta for task to get meta info to build a pod
func (tr *TaskRun) GetBuildPodMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      tr.Name,
		Namespace: tr.Namespace,
		Labels: map[string]string{
			"taskrun": tr.Name,
		},
	}
}

func init() {
	SchemeBuilder.Register(&TaskRun{}, &TaskRunList{})
}
