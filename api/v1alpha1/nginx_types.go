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

package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NginxSpec defines the desired state of Nginx
type NginxSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Replicas is the number instance of Nginx
	Replicas *int32 `json:"replicas,omitempty"`
	// Image is the base iamge of Nginx
	Image string `json:"image"`
	// Config is a reference to the NGINX config object which stores the NGINX
	// configuration file. When provided the file is mounted in NGINX container on
	// "/etc/nginx/nginx.conf".
	// +optional
	Config *ConfigRef `json:"config,omitempty"`
	// Port is the listen port of Nginx
	Port int32 `json:"port"`

	// Service to expose the nginx pod
	// +optional
	Service *NginxService `json:"service,omitempty"`

	// Template used to configure the nginx pod.
	// +optional
	PodTemplate NginxPodTemplateSpec `json:"podTemplate,omitempty"`

	// Resources requirements to be set on the NGINX container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// TLS configuration.
	// +optional
	TLS []NginxTLS `json:"tls,omitempty"`

	// ExtraFiles references to additional files into a object in the cluster.
	// These additional files will be mounted on `/etc/nginx/extra_files`.
	// +optional
	ExtraFiles *FilesRef `json:"extraFiles,omitempty"`

	// HealthcheckPath defines the endpoint used to check whether instance is
	// working or not.
	// +optional
	HealthcheckPath string `json:"healthcheckPath,omitempty"`

	// Cache allows configuring a cache volume for nginx to use.
	// +optional
	Cache NginxCacheSpec `json:"cache,omitempty"`
	// Lifecycle describes actions that should be executed when
	// some event happens to nginx container.
	// +optional
	Lifecycle *NginxLifecycle `json:"lifecycle,omitempty"`
}

type NginxService struct {
	// Type is the type of the service. Defaults to the default service type value.
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`
	// LoadBalancerIP is an optional load balancer IP for the service.
	// +optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`
	// Labels are extra labels for the service.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations are extra annotations for the service.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// ExternalTrafficPolicy defines whether external traffic will be routed to
	// node-local or cluster-wide endpoints. Defaults to the default Service
	// externalTrafficPolicy value.
	// +optional
	ExternalTrafficPolicy corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"`
	// UsePodSelector defines whether Service should automatically map the
	// endpoints using the pod's label selector. Defaults to true.
	// +optional
	UsePodSelector *bool `json:"usePodSelector,omitempty"`
}

type NginxPodTemplateSpec struct {
	// Affinity to be set on the nginx pod.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// NodeSelector to be set on the nginx pod.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Annotations are custom annotations to be set into Pod.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Labels are custom labels to be added into Pod.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// HostNetwork enabled causes the pod to use the host's network namespace.
	// +optional
	HostNetwork bool `json:"hostNetwork,omitempty"`
	// Ports is the list of ports used by nginx.
	// +optional
	Ports []corev1.ContainerPort `json:"ports,omitempty"`
	// TerminationGracePeriodSeconds defines the max duration seconds which the
	// pod needs to terminate gracefully. Defaults to pod's
	// terminationGracePeriodSeconds default value.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// PodSecurityContext configures security attributes for the nginx pod.
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// ContainerSecurityContext configures security attributes for the nginx container.
	// +optional
	ContainerSecurityContext *corev1.SecurityContext `json:"containerSecurityContext,omitempty"`

	// Volumes that will attach to nginx instances
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`
	// VolumeMounts will mount volume declared above in directories
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
	// InitContainers are executed in order prior to containers being started
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/
	// +optional
	InitContainers []corev1.Container `json:"initContainers,omitempty"`
	// Containers are executed in parallel to the main nginx container
	// +optional
	Containers []corev1.Container `json:"containers,omitempty"`
	// RollingUpdate defines params to control the desired behavior of rolling update.
	// +optional
	RollingUpdate *appsv1.RollingUpdateDeployment `json:"rollingUpdate,omitempty"`
	// Toletarion defines list of taints that pod can tolerate.
	// +optional
	Toleration []corev1.Toleration `json:"toleration,omitempty"`
	// ServiceAccountName is the name of the ServiceAccount to use to run this nginx instance.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// TopologySpreadConstraints describes how a group of pods ought to spread across topology domains.
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// ConfigRef is a reference to a config object.
type ConfigRef struct {
	// Kind of the config object. Defaults to "ConfigMap".
	Kind ConfigKind `json:"kind"`
	// Name of the ConfigMap object with "nginx.conf" key inside. It must reside
	// in the same Namespace as the Nginx resource. Required when Kind is "ConfigMap".
	//
	// It's mutually exclusive with Value field.
	// +optional
	Name string `json:"name,omitempty"`
	// Value is the raw Nginx configuration. Required when Kind is "Inline".
	//
	// It's mutually exclusive with Name field.
	// +optional
	Value string `json:"value,omitempty"`
}

type ConfigKind string

const (
	// ConfigKindConfigMap is a Kind of configuration that points to a configmap
	ConfigKindConfigMap = ConfigKind("ConfigMap")
	// ConfigKindInline is a kinda of configuration that is setup as a annotation on the Pod
	// and is inject as a file on the container using the Downward API.
	ConfigKindInline = ConfigKind("Inline")
)

// NginxStatus defines the observed state of Nginx
type NginxStatus struct {
	// CurrentReplicas is the last observed number from the NGINX object.
	CurrentReplicas int32 `json:"currentReplicas,omitempty"`
	// PodSelector is the NGINX's pod label selector.
	PodSelector string `json:"podSelector,omitempty"`

	Deployments []DeploymentStatus `json:"deployments,omitempty"`
	Services    []ServiceStatus    `json:"services,omitempty"`
}

type DeploymentStatus struct {
	// Name is the name of the Deployment created by nginx
	Name string `json:"name"`
}

type ServiceStatus struct {
	// Name is the name of the Service created by nginx
	Name      string   `json:"name"`
	IPs       []string `json:"ips,omitempty"`
	Hostnames []string `json:"hostnames,omitempty"`
}
type NginxTLS struct {
	// SecretName is the name of the Secret which contains the certificate-key
	// pair. It must reside in the same Namespace as the Nginx resource.
	//
	// NOTE: The Secret should follow the Kubernetes TLS secrets type.
	// More info: https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets.
	SecretName string `json:"secretName"`
	// Hosts are a list of hosts included in the TLS certificate. Defaults to the
	// wildcard of hosts: "*".
	// +optional
	Hosts []string `json:"hosts,omitempty"`
}

// FilesRef is a reference to arbitrary files stored into a ConfigMap in the
// cluster.
type FilesRef struct {
	// Name points to a ConfigMap resource (in the same namespace) which holds
	// the files.
	Name string `json:"name"`
	// Files maps each key entry from the ConfigMap to its relative location on
	// the nginx filesystem.
	// +optional
	Files map[string]string `json:"files,omitempty"`
}

type NginxCacheSpec struct {
	// InMemory if set to true creates a memory backed volume.
	InMemory bool `json:"inMemory,omitempty"`
	// Path is the mount path for the cache volume.
	Path string `json:"path"`
	// Size is the maximum size allowed for the cache volume.
	// +optional
	Size *resource.Quantity `json:"size,omitempty"`
}

type NginxLifecycle struct {
	PostStart *NginxLifecycleHandler `json:"postStart,omitempty"`
	PreStop   *NginxLifecycleHandler `json:"preStop,omitempty"`
}

type NginxLifecycleHandler struct {
	Exec *corev1.ExecAction `json:"exec,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Nginx is the Schema for the nginxes API
type Nginx struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NginxSpec   `json:"spec,omitempty"`
	Status NginxStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NginxList contains a list of Nginx
type NginxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nginx `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nginx{}, &NginxList{})
}
