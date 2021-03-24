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
	"strings"

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

	// retrieve the PipelineResources to which taskRun refers
	inputPRIndexedByTaskResourceName := make(map[string]pipestudiov1alpha1.PipelineResource)
	for _, resourceBinding := range taskRun.Spec.Inputs.Resources {
		pr := &pipestudiov1alpha1.PipelineResource{}
		if err = r.Get(ctx, client.ObjectKey{Namespace: taskRun.Namespace, Name: resourceBinding.ResourceRef.Name}, pr); err != nil {
			if errors.IsNotFound(err) {
				// TODO(处理TaskRun指向的资源不存在的情况)
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}
		inputPRIndexedByTaskResourceName[resourceBinding.Name] = *pr
	}

	// retireve the pod
	pod := &corev1.Pod{}
	if err = r.Get(ctx, client.ObjectKey{Namespace: taskRun.Namespace, Name: taskRun.Name}, pod); err != nil {
		r.Log.Error(err, "Failed to find pod", "taskRun.name", taskRun.Name)
		if errors.IsNotFound(err) {
			r.Log.Info("Creating a pod for TaskRun", "taskRun.name", taskRun.Name)
			newPod, err := newPodForTaskRun(taskRun, task, inputPRIndexedByTaskResourceName)
			if err != nil {
				r.Log.Error(err, "Parameters of pod is wrong", "taskRun.name", taskRun.Name)
				return ctrl.Result{}, nil
			}
			if err := ctrl.SetControllerReference(taskRun, newPod, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
			err = r.Create(ctx, newPod)
			if err != nil {
				r.Log.Error(err, "Failed to create pod", "taskRun.name", taskRun.Name)
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

func getPathFromTR(tr pipestudiov1alpha1.TaskResource, trType string) string {
	if trType == "input" {
		if tr.TargetPath == "" {
			return "/workspace/" + tr.Name
		}
		return "/workspace/" + tr.TargetPath
	}
	return "/workspace/output/" + tr.Name
}

// func for get url
func getURLFromPR(pr pipestudiov1alpha1.PipelineResource) string {
	for _, param := range pr.Spec.Params {
		if param.Name == "url" {
			return param.Value
		}
	}
	return ""
}

type params map[string]string

func replaceTaskParams(tr *pipestudiov1alpha1.TaskRun, t *pipestudiov1alpha1.Task) (result params, err error) {
	err = nil
	result = params{}
	turnToParams := func(ps []pipestudiov1alpha1.Param) (r params) {
		r = params{}
		for _, p := range ps {
			r[p.Name] = p.Value
		}
		return
	}

	taskrunInputs := turnToParams(tr.Spec.Inputs.Params)

	taskInputs := t.Spec.Inputs.Params

	for _, param := range taskInputs {
		if v, ok := taskrunInputs[param.Name]; ok {
			result[fmt.Sprintf("inputs.params.%s", param.Name)] = v
		} else if param.Default != "" {
			result[fmt.Sprintf("inputs.params.%s", param.Name)] = param.Default
		} else {
			err = fmt.Errorf("the parameter %s in task inputs does not have default value and there is no value in taskrun can be refered", param.Name)
		}
	}

	for k := range taskrunInputs {
		if _, ok := result[fmt.Sprintf("inputs.params.%s", k)]; !ok {
			err = fmt.Errorf("the parameter %s in taskrun inputs dose not have reference in task", k)
		}
	}

	return
}

func newPodForTaskRun(tr *pipestudiov1alpha1.TaskRun, t *pipestudiov1alpha1.Task, im map[string]pipestudiov1alpha1.PipelineResource) (pod *corev1.Pod, err error) {

	// add volumeMount into containers
	err = nil
	inputParams, err := replaceTaskParams(tr, t)
	if err != nil {
		return nil, err
	}

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}
	taskResourceMap := make(map[string]bool)
	// 把task的Inputs中的TaskResource作为volume绑在要创建的pod上。
	if t.Spec.Inputs != nil {
		for _, tr := range t.Spec.Inputs.Resources {
			if tr.Type == pipestudiov1alpha1.PipelineResourceTypeGit {
				volumes = append(volumes, corev1.Volume{
					Name: tr.Name,
					VolumeSource: corev1.VolumeSource{
						GitRepo: &corev1.GitRepoVolumeSource{
							Repository: getURLFromPR(im[tr.Name]),
							Directory:  ".",
						},
					},
				})
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      tr.Name,
					MountPath: getPathFromTR(tr, "input"),
				})
			}
			taskResourceMap[tr.Name] = true
		}
	}

	// 把task的Outputs中的TaskResource作为volume绑在要创建的pod上。
	if t.Spec.Outputs != nil {
		for _, tr := range t.Spec.Outputs.Resources {
			if taskResourceMap[tr.Name] {
				continue
			}
			// TODO(改成pvc，这样才能在Task之间传递)
			volumes = append(volumes, corev1.Volume{
				Name: tr.Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      tr.Name,
				MountPath: getPathFromTR(tr, "output"),
			})
		}

	}

	// mount pvc on /pvc
	if _, ok := tr.Labels["pipelinerun"]; ok {
		volumes = append(volumes, corev1.Volume{
			Name: "pvc",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: tr.Labels["pipelinerun"],
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "pvc",
			MountPath: "/pvc",
		})
	}

	containers := t.Spec.Steps

	// 用来替换arg和workingDir
	replace := func(str string) (result string, err error) {
		for {
			if index := strings.Index(str, "${"); index != -1 {
				rightBracketI := strings.Index(str, "}")
				// 找不到}退出
				if rightBracketI == -1 {
					break
				}
				if _, ok := inputParams[str[index+2:rightBracketI]]; ok {
					str = str[:index] + inputParams[str[index+2:rightBracketI]] + str[rightBracketI+1:]
				} else {
					err = fmt.Errorf("Can't find %s", str[index+2:rightBracketI])
					break
				}
			} else {
				break
			}
		}
		result = str
		return
	}

	//检查替换args和workingDir
	for i := 0; i < len(containers); i++ {
		containers[i].VolumeMounts = append(containers[i].VolumeMounts, volumeMounts...)
		containers[i].WorkingDir, err = replace(containers[i].WorkingDir)
		for j := 0; j < len(containers[i].Args); j++ {
			containers[i].Args[j], err = replace(containers[i].Args[j])
		}
	}

	serviceAccount := "default"
	if tr.Spec.ServiceAccount != "" {
		serviceAccount = tr.Spec.ServiceAccount
	}

	return &corev1.Pod{
		ObjectMeta: tr.GetBuildPodMeta(),
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount,
			Containers:         containers,
			Volumes:            append(t.Spec.Volumes, volumes...),
			RestartPolicy:      "OnFailure",
		},
	}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipestudiov1alpha1.TaskRun{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
