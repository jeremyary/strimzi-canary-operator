/*


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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	canaryv1 "github.com/jeremyary/strimzi-canary-operator/api/v1"
)

// CanaryReconciler reconciles a Canary object
type CanaryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=canary.strimzi.io,resources=canaries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=canary.strimzi.io,resources=canaries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

func (r *CanaryReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	ctx := context.Background()
	log := r.Log.WithValues("canary", req.NamespacedName)
	canary := &canaryv1.Canary{}

	err := r.Get(ctx, req.NamespacedName, canary)
	if err != nil {
		if errors.IsNotFound(err) {
			// resource not found, possibly deleted after reconcile request; return without re-queue
			log.Info("Canary resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		//Error reading object; return/requeue
		log.Error(err, "Failed to get Canary Resource")
		return ctrl.Result{}, err
	}

	// check if deployment exists; create if not found
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: canary.Name, Namespace: canary.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {

		// Define a new deployment
		dep := r.deploymentForCanary(canary, log)
		log.Info("Creating a new Deployment", "Deployment.Namespace",
			dep.Namespace, "Deployment.Name", dep.Name)

		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace",
				dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// deployment created successfully; return/requeue
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// ensure the deployment size is the same as the spec
	size := canary.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace",
				found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated; return/requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// Update Canary status w/pod names
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(canary.Namespace),
		client.MatchingLabels(labelsForCanary(canary.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Canary.Namespace", canary.Namespace,
			"Canary.Name", canary.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, canary.Status.Nodes) {
		canary.Status.Nodes = podNames
		err := r.Status().Update(ctx, canary)
		if err != nil {
			log.Error(err, "Failed to update Canary status")
			return ctrl.Result{}, err
		}
	}

	// no issues; return/requeue
	return ctrl.Result{}, nil
}

func (r *CanaryReconciler) deploymentForCanary(canary *canaryv1.Canary, log logr.Logger) *appsv1.Deployment {
	log.Info("constructing deployment...")
	ls := labelsForCanary(canary.Name)
	replicas := canary.Spec.Size

	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	for _, secretVolume := range canary.Spec.SecretVolumes {

		volumes = append(volumes, corev1.Volume{
			Name: secretVolume.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  secretVolume.SecretName,
					DefaultMode: ptrToInt32(288),
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      secretVolume.Name,
			MountPath: secretVolume.MountPath,
			ReadOnly:  true,
		})
	}
	log.Info("SecretVolumes: ", "volumes", volumes)
	log.Info("VolumeMounts: ", "volumeMounts", volumeMounts)

	envs := []corev1.EnvVar{
		{
			Name:  "KAFKA_BOOTSTRAP_URL",
			Value: canary.Spec.KafkaConfig.BootstrapUrl,
		},
		{
			Name:  "PRODUCER_TRAFFIC_TOPIC",
			Value: canary.Spec.TrafficProducer.Topic,
		},
		{
			Name:  "PRODUCER_TRAFFIC_SEND_RATE_IN_SEC",
			Value: canary.Spec.TrafficProducer.SendRate,
		},
		{
			Name:  "PRODUCER_CLIENT_ID",
			Value: canary.Spec.TrafficProducer.ClientId,
		},
	}
	log.Info("EnvVars: ", "EnvVars", envs)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canary.Name,
			Namespace: canary.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:         "strimzi-canary",
						Image:        "quay.io/jary/go-stoker:latest",
						VolumeMounts: volumeMounts,
						Env:          envs,
					}},
					Volumes: volumes,
				},
			},
		},
	}
	log.Info("Deployment: ", "deployment", deployment)

	// Set Canary instance as owner & controller
	ctrl.SetControllerReference(canary, deployment, r.Scheme)
	return deployment
}

func labelsForCanary(name string) map[string]string {
	return map[string]string{"app": "strimzi-canary", "canary_cr": name}
}

func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func (r *CanaryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&canaryv1.Canary{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func ptrToInt32(i int32) *int32 {
	return &i
}
