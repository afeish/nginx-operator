/*
Copyright 2024 neocxf.

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

package controller

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	"github.com/neocxf/nginx-operator/api/v1alpha1"
	"github.com/neocxf/nginx-operator/pkg/k8s"
	"github.com/pingcap/errors"
)

// NginxReconciler reconciles a Nginx object
type NginxReconciler struct {
	client.Client
	Log logr.Logger

	Scheme           *runtime.Scheme
	Recorder         record.EventRecorder
	AnnotationFilter labels.Selector
}

//+kubebuilder:rbac:groups=nginx.example.org,resources=nginxes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nginx.example.org,resources=nginxes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nginx.example.org,resources=nginxes/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Nginx object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *NginxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	log := r.Log.WithValues("nginx", req.NamespacedName)

	var instance v1alpha1.Nginx
	err := r.Client.Get(ctx, req.NamespacedName, &instance)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Nginx resource not found, skipping reconcile")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Unable to get Nginx resource")
		return ctrl.Result{}, err
	}

	if !r.shouldManageNginx(&instance) {
		log.V(1).Info("Nginx resource doesn't match annotations filters, skipping it")
		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Minute}, nil
	}

	if err := r.reconcileNginx(ctx, &instance); err != nil {
		log.Error(err, "Fail to reconcile")
		return ctrl.Result{}, err
	}

	if err := r.refreshStatus(ctx, &instance); err != nil {
		log.Error(err, "Fail to refresh status subresource")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NginxReconciler) reconcileNginx(ctx context.Context, nginx *v1alpha1.Nginx) error {
	if err := r.reconcileDeployment(ctx, nginx); err != nil {
		return err
	}

	return nil
}

func (r *NginxReconciler) reconcileDeployment(ctx context.Context, nginx *v1alpha1.Nginx) error {
	newDeploy, err := k8s.NewDeployment(nginx)
	if err != nil {
		return fmt.Errorf("failed to build Deployment from Nginx: %w", err)
	}

	var currentDeploy appsv1.Deployment
	err = r.Client.Get(ctx, types.NamespacedName{Name: newDeploy.Name, Namespace: newDeploy.Namespace}, &currentDeploy)
	if errors.IsNotFound(err) {
		return r.Client.Create(ctx, newDeploy)
	}

	if err != nil {
		return fmt.Errorf("failed to retrieve Deployment: %w", err)
	}

	existingNginxSpec, err := k8s.ExtractNginxSpec(currentDeploy.ObjectMeta)
	if err != nil {
		return fmt.Errorf("failed to extract Nginx spec from Deployment annotations: %w", err)
	}

	if reflect.DeepEqual(nginx.Spec, existingNginxSpec) {
		return nil
	}

	replicas := currentDeploy.Spec.Replicas

	patch := client.StrategicMergeFrom(currentDeploy.DeepCopy())
	currentDeploy.Spec = newDeploy.Spec

	if newDeploy.Spec.Replicas == nil {
		// NOTE: replicas field is set to nil whenever it's managed by some
		// autoscaler controller e.g HPA.
		currentDeploy.Spec.Replicas = replicas
	}

	err = k8s.SetNginxSpec(&currentDeploy.ObjectMeta, nginx.Spec)
	if err != nil {
		return fmt.Errorf("failed to set Nginx spec in Deployment annotations: %w", err)
	}

	err = r.Client.Patch(ctx, &currentDeploy, patch)
	if err != nil {
		return fmt.Errorf("failed to patch Deployment: %w", err)
	}

	return nil
}

func (r *NginxReconciler) shouldManageNginx(nginx *v1alpha1.Nginx) bool {
	// empty filter matches all resources
	if r.AnnotationFilter == nil || r.AnnotationFilter.Empty() {
		return true
	}

	return r.AnnotationFilter.Matches(labels.Set(nginx.Annotations))
}

func (r *NginxReconciler) refreshStatus(ctx context.Context, nginx *v1alpha1.Nginx) error {
	deploys, err := listDeployments(ctx, r.Client, nginx)
	if err != nil {
		return err
	}

	var deployStatuses []v1alpha1.DeploymentStatus
	var replicas int32
	for _, d := range deploys {
		replicas += d.Status.Replicas
		deployStatuses = append(deployStatuses, v1alpha1.DeploymentStatus{Name: d.Name})
	}

	services, err := listServices(ctx, r.Client, nginx)
	if err != nil {
		return fmt.Errorf("failed to list services for nginx: %v", err)
	}

	sort.Slice(nginx.Status.Services, func(i, j int) bool {
		return nginx.Status.Services[i].Name < nginx.Status.Services[j].Name
	})

	status := v1alpha1.NginxStatus{
		CurrentReplicas: replicas,
		PodSelector:     k8s.LabelsForNginxString(nginx.Name),
		Deployments:     deployStatuses,
		Services:        services,
	}

	if reflect.DeepEqual(nginx.Status, status) {
		return nil
	}

	nginx.Status = status

	err = r.Client.Status().Update(ctx, nginx)
	if err != nil {
		return fmt.Errorf("failed to update nginx status: %v", err)
	}

	return nil
}

func listDeployments(ctx context.Context, c client.Client, nginx *v1alpha1.Nginx) ([]appsv1.Deployment, error) {
	var deployList appsv1.DeploymentList

	err := c.List(ctx, &deployList, &client.ListOptions{
		Namespace:     nginx.Namespace,
		LabelSelector: labels.SelectorFromSet(k8s.LabelsForNginx(nginx.Name)),
	})
	if err != nil {
		return nil, err
	}

	deploys := deployList.Items

	// NOTE: specific implementation for backward compatibility w/ Deployments
	// that does not have Nginx labels yet.
	if len(deploys) == 0 {
		err = c.List(ctx, &deployList, &client.ListOptions{Namespace: nginx.Namespace})
		if err != nil {
			return nil, err
		}

		desired := *metav1.NewControllerRef(nginx, schema.GroupVersionKind{
			Group:   v1alpha1.GroupVersion.Group,
			Version: v1alpha1.GroupVersion.Version,
			Kind:    "Nginx",
		})

		for _, deploy := range deployList.Items {
			for _, owner := range deploy.OwnerReferences {
				if reflect.DeepEqual(owner, desired) {
					deploys = append(deploys, deploy)
				}
			}
		}
	}

	sort.Slice(deploys, func(i, j int) bool { return deploys[i].Name < deploys[j].Name })

	return deploys, nil
}

// listServices return all the services for the given nginx sorted by name
func listServices(ctx context.Context, c client.Client, nginx *v1alpha1.Nginx) ([]v1alpha1.ServiceStatus, error) {
	serviceList := &corev1.ServiceList{}
	labelSelector := labels.SelectorFromSet(k8s.LabelsForNginx(nginx.Name))
	listOps := &client.ListOptions{Namespace: nginx.Namespace, LabelSelector: labelSelector}
	err := c.List(ctx, serviceList, listOps)
	if err != nil {
		return nil, err
	}

	var services []v1alpha1.ServiceStatus
	for _, s := range serviceList.Items {
		svc := v1alpha1.ServiceStatus{
			Name: s.Name,
		}

		for _, ingStatus := range s.Status.LoadBalancer.Ingress {
			if ingStatus.IP != "" {
				svc.IPs = append(svc.IPs, ingStatus.IP)
			}

			if ingStatus.Hostname != "" {
				svc.Hostnames = append(svc.Hostnames, ingStatus.Hostname)
			}
		}

		slices.Sort(svc.IPs)
		slices.Sort(svc.Hostnames)

		services = append(services, svc)
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NginxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Nginx{}).
		Owns(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
