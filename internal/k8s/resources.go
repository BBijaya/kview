package k8s

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Resource represents a Kubernetes resource with common metadata
type Resource struct {
	UID         string
	APIVersion  string
	Kind        string
	Namespace   string
	Name        string
	ClusterID   string
	Labels      map[string]string
	Annotations map[string]string
	OwnerRefs   []OwnerReference
	Conditions  []Condition
	Raw         *unstructured.Unstructured
	FetchedAt   time.Time
}

// OwnerReference represents an owner reference to another resource
type OwnerReference struct {
	Kind string
	Name string
	UID  string
}

// Condition represents a resource condition
type Condition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// WatchEvent represents a resource watch event
type WatchEvent struct {
	Type     WatchEventType
	Resource *Resource
}

// WatchEventType is the type of watch event
type WatchEventType string

const (
	WatchEventAdded    WatchEventType = "ADDED"
	WatchEventModified WatchEventType = "MODIFIED"
	WatchEventDeleted  WatchEventType = "DELETED"
	WatchEventError    WatchEventType = "ERROR"
)

// PodInfo contains pod-specific information
type PodInfo struct {
	Resource
	Phase              string
	Ready              string
	Restarts           int32
	Age                time.Duration
	NodeName           string
	IP                 string
	Containers         []ContainerInfo
	InitContainers     []ContainerInfo
	CPURequest         int64 // total nanocores (sum across all containers)
	CPULimit           int64 // total nanocores
	MemRequest         int64 // total bytes
	MemLimit           int64 // total bytes
	HostNetwork        bool
	ServiceAccountName string   // Pod's service account
	VolumeSecrets      []string // Secret names from volumes
	VolumeConfigMaps   []string // ConfigMap names from volumes
	VolumePVCs         []string // PVC names from volumes
}

// PodMetrics contains per-pod CPU/memory usage from metrics-server
type PodMetrics struct {
	Namespace string
	Name      string
	CPUUsage  int64 // nanocores
	MemUsage  int64 // bytes
}

// ContainerPort represents a container port definition
type ContainerPort struct {
	Name          string
	ContainerPort int32
	Protocol      string
}

// ContainerInfo contains container-specific information
type ContainerInfo struct {
	Name              string
	Image             string
	Ready             bool
	RestartCount      int32
	State             string
	StateReason       string
	StateMessage      string
	LastTerminatedAt  time.Time // When the previous container instance terminated (restart time)
	CPURequest        int64     // nanocores
	CPULimit          int64 // nanocores
	MemRequest        int64 // bytes
	MemLimit          int64 // bytes
	Ports             []ContainerPort
	HasLivenessProbe  bool
	HasReadinessProbe bool
	Privileged        bool
	RunAsNonRoot      *bool // nil = not set, true = enforced, false = explicitly root
	EnvRefSecrets     []string // Secret names referenced via env/envFrom
	EnvRefConfigMaps  []string // ConfigMap names referenced via env/envFrom
}

// DeploymentInfo contains deployment-specific information
type DeploymentInfo struct {
	Resource
	Replicas          int32
	ReadyReplicas     int32
	UpdatedReplicas   int32
	AvailableReplicas int32
	Age               time.Duration
	Strategy          string
}

// ServiceInfo contains service-specific information
type ServiceInfo struct {
	Resource
	Type        string
	ClusterIP   string
	ExternalIP  string
	Ports       []ServicePort
	Age         time.Duration
	Selector    map[string]string
}

// ServicePort represents a service port
type ServicePort struct {
	Name       string
	Port       int32
	TargetPort string
	Protocol   string
	NodePort   int32
}

// EndpointInfo contains endpoint-specific information
type EndpointInfo struct {
	Resource
	Endpoints string        // Pre-formatted ip:port string for display
	Age       time.Duration
}

// LogOptions contains options for log retrieval
type LogOptions struct {
	Container    string
	Follow       bool
	Previous     bool
	TailLines    int64
	SinceSeconds int64
	Timestamps   bool
}

// ConfigMapInfo contains configmap-specific information
type ConfigMapInfo struct {
	Resource
	DataCount int
	Age       time.Duration
}

// SecretInfo contains secret-specific information
type SecretInfo struct {
	Resource
	Type      string
	DataCount int
	Age       time.Duration
}

// IngressInfo contains ingress-specific information
type IngressInfo struct {
	Resource
	Class   string
	Hosts   []string
	Address string
	Ports   string
	Age     time.Duration
	Rules   []IngressRule
}

// IngressRule represents an ingress rule
type IngressRule struct {
	Host  string
	Paths []IngressPath
}

// IngressPath represents an ingress path
type IngressPath struct {
	Path        string
	PathType    string
	ServiceName string
	ServicePort string
}

// PVCInfo contains PVC-specific information
type PVCInfo struct {
	Resource
	Status       string
	Volume       string
	Capacity     string
	AccessModes  []string
	StorageClass string
	Age          time.Duration
}

// StatefulSetInfo contains statefulset-specific information
type StatefulSetInfo struct {
	Resource
	Replicas      int32
	ReadyReplicas int32
	Age           time.Duration
	ServiceName   string
}

// NamespaceInfo contains namespace-specific information
type NamespaceInfo struct {
	Name   string
	Status string // Active, Terminating
	Age    time.Duration
}

// NodeInfo contains node-specific information
type NodeInfo struct {
	Resource
	Status           string   // Ready, NotReady, Unknown
	Roles            []string // control-plane, worker, etc.
	Taints           []string // NoSchedule, NoExecute, etc.
	Age              time.Duration
	Version          string // Kubelet version
	InternalIP       string
	ExternalIP       string
	OS               string // linux, windows
	Arch             string // amd64, arm64
	PodCount         int    // Number of pods running on this node
	CPUAllocatable   int64  // Allocatable CPU in nanocores
	MemAllocatable   int64  // Allocatable memory in bytes
}

// NodeMetrics contains per-node CPU and memory usage from metrics-server
type NodeMetrics struct {
	Name     string
	CPUUsage int64 // nanocores
	MemUsage int64 // bytes
}

// EventInfo contains event-specific information
type EventInfo struct {
	Resource
	Type       string // Normal, Warning
	Reason     string
	Message    string
	ObjectKind string
	ObjectName string
	Count      int32
	FirstSeen  time.Time
	LastSeen   time.Time
	Age        time.Duration
}

// ReplicaSetInfo contains replicaset-specific information
type ReplicaSetInfo struct {
	Resource
	DesiredReplicas   int32
	ReadyReplicas     int32
	AvailableReplicas int32
	Age               time.Duration
	OwnerKind         string
	OwnerName         string
}

// DaemonSetInfo contains daemonset-specific information
type DaemonSetInfo struct {
	Resource
	DesiredNumber   int32
	CurrentNumber   int32
	ReadyNumber     int32
	AvailableNumber int32
	NodeSelector    string
	Age             time.Duration
}

// JobInfo contains job-specific information
type JobInfo struct {
	Resource
	Completions int32
	Succeeded   int32
	Failed      int32
	Active      int32
	Duration    time.Duration
	Age         time.Duration
	Status      string // Running, Complete, Failed
}

// CronJobInfo contains cronjob-specific information
type CronJobInfo struct {
	Resource
	Schedule     string
	Suspend      bool
	Active       int32
	LastSchedule time.Time
	Age          time.Duration
}

// HPAInfo contains HorizontalPodAutoscaler-specific information
type HPAInfo struct {
	Resource
	Reference      string // e.g. "Deployment/nginx"
	Targets        string // e.g. "50%/80%" or "<unknown>/80%"
	MinReplicas    int32
	MaxReplicas    int32
	CurrentReplicas int32
	Age            time.Duration
}

// PVInfo contains PersistentVolume-specific information
type PVInfo struct {
	Resource
	Capacity     string
	AccessModes  []string
	ReclaimPolicy string
	Status       string // Available, Bound, Released, Failed
	Claim        string // namespace/name of bound PVC
	StorageClass string
	Reason       string
	Age          time.Duration
}

// RoleBindingInfo contains RoleBinding/ClusterRoleBinding-specific information
type RoleBindingInfo struct {
	Resource
	RoleKind string // "Role" or "ClusterRole"
	RoleName string
	Subjects string // comma-separated subject descriptions
	Age      time.Duration
}

// HelmReleaseInfo contains Helm release information extracted from Helm 3 Secrets
type HelmReleaseInfo struct {
	Resource
	Chart        string // chart name (e.g. "nginx")
	ChartVersion string // chart version (e.g. "15.0.0")
	AppVersion   string // app version from chart metadata
	Status       string // deployed, failed, superseded, pending-install, etc.
	Revision     int    // release revision number
	Age          time.Duration
}
