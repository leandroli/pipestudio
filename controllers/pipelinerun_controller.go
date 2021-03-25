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
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipestudiov1alpha1 "github.com/leandroli/pipestudio/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineRunReconciler reconciles a PipelineRun object
type PipelineRunReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pipestudio.github.com,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipestudio.github.com,resources=pipelineruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pipestudio.github.com,resources=pipelineruns/finalizers,verbs=update
// +kubebuilder:rbac:groups=pipestudio.github.com,resources=taskruns,verbs=get;list;watch;creat;update;patch;delete
// +kubebuilder:rbac:groups=pipestudio.github.com,resources=taskruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1,resources=persistentvolumeclaims,verbs=get;list;watch;creat;update;patch;delete
// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PipelineRun object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *PipelineRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("pipelinerun", req.NamespacedName)

	// retrieve a pipelineRun instance
	instance := &pipestudiov1alpha1.PipelineRun{}
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
	pipelineRun := instance

	// retrieve the pipeline to which piplineRun refer
	pipeline := &pipestudiov1alpha1.Pipeline{}
	err = r.Get(ctx,
		client.ObjectKey{
			Name:      pipelineRun.Spec.PipelineRef.Name,
			Namespace: pipelineRun.Namespace,
		},
		pipeline,
	)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// retrieve pvc to mount on the pods
	pvc := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx,
		client.ObjectKey{
			Name:      pipelineRun.Name,
			Namespace: pipelineRun.Namespace,
		},
		pvc,
	)
	if err != nil {
		if errors.IsNotFound(err) {
			newPvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pipelineRun.Name,
					Namespace: pipelineRun.Namespace,
					Labels: map[string]string{
						"pipelinerun": pipelineRun.Name,
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1G"),
						},
					},
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				},
			}
			if err := ctrl.SetControllerReference(pipelineRun, newPvc, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, newPvc); err != nil {
				r.Log.Error(err, "Fail to create pvc", "PipelineRun.name", pipelineRun.Name)
				requeueAfter, _ := time.ParseDuration("1m")
				return ctrl.Result{Requeue: true, RequeueAfter: requeueAfter}, err
			}
			r.Log.Info("a pvc has been created", "pvc", newPvc.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// update pipelineRun status
	// list the taskRun related to this pipelineRun. if there is no taskrun, create one
	taskRunList := &pipestudiov1alpha1.TaskRunList{}
	lbs := map[string]string{
		"pipelinerun": pipelineRun.Name,
	}
	labelSelector := labels.SelectorFromSet(lbs)
	if err := r.List(
		ctx, taskRunList,
		&client.ListOptions{
			Namespace:     pipelineRun.Namespace,
			LabelSelector: labelSelector,
		},
	); err != nil {
		if errors.IsNotFound(err) {
			pipelineRun.Status = pipestudiov1alpha1.PipelineRunStatus{
				TaskRuns: nil,
			}
			if err := r.Status().Update(ctx, pipelineRun); err != nil {
				r.Log.Error(err, "Failed to update PipelineRun status")
				return ctrl.Result{}, err
			}
			r.Log.Info("Succeed to update pipelineRun status")
		}
	}
	statuses := []pipestudiov1alpha1.PipelineTaskRunStatus{}
	for _, taskRun := range taskRunList.Items {
		statuses = append(
			statuses,
			pipestudiov1alpha1.PipelineTaskRunStatus{
				Name:          taskRun.Name,
				TaskRunStatus: taskRun.Status,
			})
	}
	if !reflect.DeepEqual(pipelineRun.Status.TaskRuns, statuses) {
		pipelineRun.Status.TaskRuns = statuses
		if err = r.Status().Update(ctx, pipelineRun); err != nil {
			r.Log.Error(err, "Failed to update PipelineRun status")
			return ctrl.Result{}, err
		}
	}

	// execute tasks sequentially in the order of declaration in Pipeline
	// check the status of taskrun and execute next one if it is exited whit 0
	pipelineTasks := pipeline.Spec.Tasks
	for i := 0; i < len(pipelineTasks); i++ {
		pipelineTask := pipelineTasks[i]
		key := client.ObjectKey{
			Namespace: pipelineRun.Namespace,
			Name:      pipelineRun.Name + "-" + pipelineTask.Name,
		}
		taskRun := &pipestudiov1alpha1.TaskRun{}
		if err := r.Get(ctx, key, taskRun); err != nil {
			if errors.IsNotFound(err) {
				newTaskRun, err := newTaskRunforPipelineTask(&pipelineTask, pipelineRun)
				if err != nil {
					r.Log.Error(err, "Fail to generate TaskRun", "PipelineTask.name", pipelineTask.Name)
					return ctrl.Result{}, nil
				}
				annotations := make(map[string]string)
				if i != 0 {
					annotations["frontTaskRun"] = pipelineRun.Name + "-" + pipelineTasks[i-1].Name
				}
				if i != len(pipelineTasks)-1 {
					annotations["nextTaskRun"] = pipelineRun.Name + "-" + pipelineTasks[i+1].Name
				}
				if err = ctrl.SetControllerReference(pipelineRun, newTaskRun, r.Scheme); err != nil {
					return ctrl.Result{}, err
				}
				if err = r.Create(ctx, newTaskRun); err != nil {
					r.Log.Error(err, "Fail to create TaskRun", "TaskRun.name", newTaskRun.Name)
					requeueAfter, _ := time.ParseDuration("1m")
					return ctrl.Result{Requeue: true, RequeueAfter: requeueAfter}, err
				}
				r.Log.Info("a TaskRun has been created", "PipelineTask.name", pipelineTask.Name, "TaskRun.name", newTaskRun.Name)
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}
		if taskRun.Status.Steps == nil {
			return ctrl.Result{}, err
		} else {
			for _, containerStatus := range taskRun.Status.Steps {
				if containerStatus.State.Terminated == nil || containerStatus.State.Terminated.ExitCode != 0 {
					return ctrl.Result{}, err
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

func newTaskRunforPipelineTask(
	pt *pipestudiov1alpha1.PipelineTask,
	pr *pipestudiov1alpha1.PipelineRun,
) (taskrun *pipestudiov1alpha1.TaskRun, err error) {

	findResourceRef := func(name string) (pipestudiov1alpha1.PipelineResourceRef, error) {
		for _, resource := range pr.Spec.Resources {
			if resource.Name == name {
				return resource.ResourceRef, nil
			}
		}
		return pipestudiov1alpha1.PipelineResourceRef{},
			fmt.Errorf("can't find resource %s in PipelineRun", name)
	}

	taskRunInputs := &pipestudiov1alpha1.TaskRunInputs{}
	for _, resource := range pt.Inputs.Resources {
		resourceRef, err := findResourceRef(resource.Resource)
		if err != nil {
			return nil, err
		}
		taskRunInputs.Resources = append(taskRunInputs.Resources, pipestudiov1alpha1.TaskResourceBinding{
			Name:        resource.Name,
			ResourceRef: resourceRef,
		})
	}
	taskRunInputs.Params = pt.Inputs.Params
	return &pipestudiov1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pr.Name + "-" + pt.Name,
			Namespace: pr.Namespace,
			Labels: map[string]string{
				"pipelinerun": pr.Name,
			},
		},
		Spec: pipestudiov1alpha1.TaskRunSpec{
			ServiceAccount: pr.Spec.ServiceAccount,
			TaskRef:        pt.TaskRef,
			Inputs:         taskRunInputs,
		},
	}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipestudiov1alpha1.PipelineRun{}).
		Owns(&pipestudiov1alpha1.TaskRun{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
