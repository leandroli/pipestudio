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

package controllers

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipestudiov1alpha1 "github.com/leandroli/pipestudio/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// TaskRunReconciler reconciles a TaskRun object
type TaskRunReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pipestudio.github.com,resources=taskruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipestudio.github.com,resources=taskruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pipestudio.github.com,resources=taskruns/finalizers,verbs=update
// +kubebuilder:rbac:groups=pipestudio.github.com,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TaskRun object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *TaskRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("taskrun", req.NamespacedName)

	// your logic here

	// retireve a TaskRun instance
	instance := &pipestudiov1alpha1.TaskRun{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// retireve the task that the taskRun refers to
	taskRun := instance
	task := &pipestudiov1alpha1.Task{}
	if err = r.Get(ctx, client.ObjectKey{Namespace: taskRun.Namespace, Name: taskRun.Spec.TaskRef.Name}, task); err != nil {
		if errors.IsNotFound(err) {
			// TODO(处理有TaskRun但没有Task的情况, deal with the scenario that Task to which TaskRun refers dose not exist)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// retireve the pod
	pod := &corev1.Pod{}
	if err = r.Get(ctx, client.ObjectKey{Namespace: taskRun.Namespace, Name: taskRun.Name}, pod); err != nil {
		r.Log.Error(err, "Failed to find pod", "taskRun.name", taskRun.Name)
		if errors.IsNotFound(err) {
			r.Log.Info("Creating a pod for TaskRun", "taskRun.name", taskRun.Name)
			newPod := newPodForTaskRun(taskRun, task)
			if err := ctrl.SetControllerReference(taskRun, newPod, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
			err = r.Create(ctx, newPod)
			if err != nil {
				r.Log.Error(err, "Failed to create pod", "pod.name", newPod.Name)
				return ctrl.Result{}, err
			}
			r.Log.Info("A pod has been created for TaskRun", "taskRun.name", taskRun.Name)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	// update status if necessary
	status := pipestudiov1alpha1.TaskRunStatus{
		PodName: pod.Name,
		Steps:   pod.Status.ContainerStatuses,
	}
	if !reflect.DeepEqual(status, taskRun.Status) {
		taskRun.Status = status
		err := r.Status().Update(ctx, taskRun)
		if err != nil {
			r.Log.Error(err, "Failed to update TaskRun status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func newPodForTaskRun(tr *pipestudiov1alpha1.TaskRun, t *pipestudiov1alpha1.Task) *corev1.Pod {
	containers := t.Spec.Steps
	for i := 0; i < len(containers); i++ {
		containers[i].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "workspace",
				MountPath: "/workspace",
			},
		}
	}
	return &corev1.Pod{
		ObjectMeta: tr.GetBuildPodMeta(),
		Spec: corev1.PodSpec{
			Containers: containers,
			Volumes: []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipestudiov1alpha1.TaskRun{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
