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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipestudiov1alpha1 "github.com/leandroli/pipestudio/api/v1alpha1"
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
// +kubebuilder:rbac:groups=pipestudio.github.com,resources=tasks,verbs=get;list;watch;creat;update;patch;delete
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
	err = r.Get(ctx, client.ObjectKey{Name: pipelineRun.Spec.PipelineRef.Name, Namespace: pipelineRun.Namespace}, pipeline)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// execute tasks sequentially in the order of declaration in Pipeline
	pipelineTasks := pipeline.Spec.Tasks
	for _, pipelineTask := range pipelineTasks {
		// 给每个一pipelineTask创建一个TaskRun，监控这个taskRun对应的pod，正常结束后执行下一个
		taskrun, err := newTaskRunforPipelineTask(&pipelineTask, pipelineRun)
		if err != nil {
			r.Log.Error(err, "Fail to generate taskRun", "pipelineTask.name", pipelineTask.Name)
			return ctrl.Result{}, nil
		}
		if err = ctrl.SetControllerReference(pipelineRun, taskrun, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err = r.Create(ctx, taskrun); err != nil {
			r.Log.Error(err, "Fail to create taskrun", "taskrun.name", taskrun.Name)
			return ctrl.Result{Requeue: false}, err
		}
		r.Log.Info("a TaskRun has been created", "pipelineTask.name", pipelineTask.Name, "taskrun.name", taskrun.Name)
		

		//
	}

	return ctrl.Result{}, nil
}


func newTaskRunforPipelineTask(pt *pipestudiov1alpha1.PipelineTask,
	pr *pipestudiov1alpha1.PipelineRun) (taskrun *pipestudiov1alpha1.TaskRun, err error) {
	//
	
	findResourceRef := func (name string) (pipestudiov1alpha1.PipelineResourceRef, error) {
		for _, resource := range pr.Spec.Resources {
			if resource.Name == name {
				return resource.ResourceRef, nil
			}
		}
		return pipestudiov1alpha1.PipelineResourceRef{}, fmt.Errorf("can't find resource %s in PipelineRun", name)
	} 

	taskRunInputs := &pipestudiov1alpha1.TaskRunInputs{}
	for _, resource := range pt.Inputs.Resources {
		resourceRef, err := findResourceRef(resource.Resource)
		if err != nil {
			return nil, err
		}
		taskRunInputs.Resources = append(taskRunInputs.Resources, pipestudiov1alpha1.TaskResourceBinding{
			Name: resource.Name,
			ResourceRef: resourceRef,
		})
	}
	taskRunInputs.Params = pt.Inputs.Params
	return &pipestudiov1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pr.Name + "-" + pt.Name,
			Namespace: pr.Namespace,
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
		Complete(r)
}
