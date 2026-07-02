package k8s

import (
	"reflect"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func testClient() *K8sClient {
	return &K8sClient{clusterID: "test-cluster"}
}

func int32Ptr(i int32) *int32 { return &i }

func TestPodToPodInfoStatus(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name          string
		pod           *corev1.Pod
		wantPhase     string
		wantReady     string
		wantRestarts  int32
		wantCtrState  string // state of first container, "" to skip
		wantCtrReason string // state reason of first container
	}{
		{
			name: "running pod one of two ready",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}, {Name: "sidecar"}},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:  "app",
							Ready: true,
							State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
						},
						{
							Name:  "sidecar",
							Ready: false,
							State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
						},
					},
				},
			},
			wantPhase:    "Running",
			wantReady:    "1/2",
			wantRestarts: 0,
			wantCtrState: "Running",
		},
		{
			name: "crashloop backoff container",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "crashy", Namespace: "default"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "app",
							Ready:        false,
							RestartCount: 7,
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason:  "CrashLoopBackOff",
									Message: "back-off 5m0s restarting failed container",
								},
							},
						},
					},
				},
			},
			wantPhase:     "Running",
			wantReady:     "0/1",
			wantRestarts:  7,
			wantCtrState:  "Waiting",
			wantCtrReason: "CrashLoopBackOff",
		},
		{
			name: "succeeded pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "batch", Namespace: "default"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "job"}},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:  "job",
							Ready: false,
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{Reason: "Completed"},
							},
						},
					},
				},
			},
			wantPhase:     "Succeeded",
			wantReady:     "0/1",
			wantRestarts:  0,
			wantCtrState:  "Terminated",
			wantCtrReason: "Completed",
		},
		{
			name: "terminating pod overrides phase",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "doomed",
					Namespace:         "default",
					DeletionTimestamp: &now,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:  "app",
							Ready: true,
							State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
						},
					},
				},
			},
			wantPhase:    "Terminating",
			wantReady:    "1/1",
			wantRestarts: 0,
			wantCtrState: "Running",
		},
		{
			name: "restart counts summed across containers",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "restarty", Namespace: "default"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "a"}, {Name: "b"}},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "a",
							Ready:        true,
							RestartCount: 3,
							State:        corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
						},
						{
							Name:         "b",
							Ready:        true,
							RestartCount: 2,
							State:        corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
						},
					},
				},
			},
			wantPhase:    "Running",
			wantReady:    "2/2",
			wantRestarts: 5,
			wantCtrState: "Running",
		},
	}

	c := testClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := c.podToPodInfo(tt.pod)

			if info.Phase != tt.wantPhase {
				t.Errorf("Phase = %q, want %q", info.Phase, tt.wantPhase)
			}
			if info.Ready != tt.wantReady {
				t.Errorf("Ready = %q, want %q", info.Ready, tt.wantReady)
			}
			if info.Restarts != tt.wantRestarts {
				t.Errorf("Restarts = %d, want %d", info.Restarts, tt.wantRestarts)
			}
			if tt.wantCtrState != "" {
				if len(info.Containers) == 0 {
					t.Fatalf("Containers is empty, want at least one")
				}
				if info.Containers[0].State != tt.wantCtrState {
					t.Errorf("Containers[0].State = %q, want %q", info.Containers[0].State, tt.wantCtrState)
				}
				if info.Containers[0].StateReason != tt.wantCtrReason {
					t.Errorf("Containers[0].StateReason = %q, want %q", info.Containers[0].StateReason, tt.wantCtrReason)
				}
			}
		})
	}
}

func TestPodToPodInfoMetadata(t *testing.T) {
	c := testClient()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:               "pod-uid-1",
			Name:              "web",
			Namespace:         "prod",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Hour)),
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: "web-abc123", UID: "rs-uid-1"},
			},
		},
		Spec: corev1.PodSpec{
			NodeName:   "node-1",
			Containers: []corev1.Container{{Name: "app"}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.5",
		},
	}

	info := c.podToPodInfo(pod)

	if info.UID != "pod-uid-1" {
		t.Errorf("UID = %q, want %q", info.UID, "pod-uid-1")
	}
	if info.Kind != "Pod" {
		t.Errorf("Kind = %q, want %q", info.Kind, "Pod")
	}
	if info.Namespace != "prod" || info.Name != "web" {
		t.Errorf("Namespace/Name = %q/%q, want prod/web", info.Namespace, info.Name)
	}
	if info.ClusterID != "test-cluster" {
		t.Errorf("ClusterID = %q, want %q", info.ClusterID, "test-cluster")
	}
	if info.NodeName != "node-1" {
		t.Errorf("NodeName = %q, want %q", info.NodeName, "node-1")
	}
	if info.IP != "10.0.0.5" {
		t.Errorf("IP = %q, want %q", info.IP, "10.0.0.5")
	}
	if info.Age <= 0 {
		t.Errorf("Age = %v, want > 0", info.Age)
	}
	wantOwners := []OwnerReference{{Kind: "ReplicaSet", Name: "web-abc123", UID: "rs-uid-1"}}
	if !reflect.DeepEqual(info.OwnerRefs, wantOwners) {
		t.Errorf("OwnerRefs = %+v, want %+v", info.OwnerRefs, wantOwners)
	}
}

func TestPodToPodInfoVolumeAndEnvRefs(t *testing.T) {
	c := testClient()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
		Spec: corev1.PodSpec{
			ServiceAccountName: "web-sa",
			Volumes: []corev1.Volume{
				{
					Name: "tls",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: "tls-secret"},
					},
				},
				{
					Name: "conf",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
						},
					},
				},
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "data-pvc",
						},
					},
				},
				{
					Name:         "scratch",
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				},
			},
			Containers: []corev1.Container{
				{
					Name: "app",
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "env-secret"},
							},
						},
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "env-config"},
							},
						},
					},
					Env: []corev1.EnvVar{
						{
							Name: "DB_PASSWORD",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "db-secret"},
									Key:                  "password",
								},
							},
						},
						{
							// Duplicate secret ref — should be deduplicated
							Name: "DB_USER",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "db-secret"},
									Key:                  "user",
								},
							},
						},
						{
							Name: "LOG_LEVEL",
							ValueFrom: &corev1.EnvVarSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "log-config"},
									Key:                  "level",
								},
							},
						},
						{Name: "PLAIN", Value: "value"},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: true,
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
		},
	}

	info := c.podToPodInfo(pod)

	if info.ServiceAccountName != "web-sa" {
		t.Errorf("ServiceAccountName = %q, want %q", info.ServiceAccountName, "web-sa")
	}
	if want := []string{"tls-secret"}; !reflect.DeepEqual(info.VolumeSecrets, want) {
		t.Errorf("VolumeSecrets = %v, want %v", info.VolumeSecrets, want)
	}
	if want := []string{"app-config"}; !reflect.DeepEqual(info.VolumeConfigMaps, want) {
		t.Errorf("VolumeConfigMaps = %v, want %v", info.VolumeConfigMaps, want)
	}
	if want := []string{"data-pvc"}; !reflect.DeepEqual(info.VolumePVCs, want) {
		t.Errorf("VolumePVCs = %v, want %v", info.VolumePVCs, want)
	}

	if len(info.Containers) != 1 {
		t.Fatalf("len(Containers) = %d, want 1", len(info.Containers))
	}
	ctr := info.Containers[0]
	if want := []string{"env-secret", "db-secret"}; !reflect.DeepEqual(ctr.EnvRefSecrets, want) {
		t.Errorf("EnvRefSecrets = %v, want %v", ctr.EnvRefSecrets, want)
	}
	if want := []string{"env-config", "log-config"}; !reflect.DeepEqual(ctr.EnvRefConfigMaps, want) {
		t.Errorf("EnvRefConfigMaps = %v, want %v", ctr.EnvRefConfigMaps, want)
	}
}

func TestDeploymentToDeploymentInfo(t *testing.T) {
	tests := []struct {
		name          string
		dep           *appsv1.Deployment
		wantReplicas  int32
		wantReady     int32
		wantUpdated   int32
		wantAvailable int32
		wantStrategy  string
	}{
		{
			name: "explicit replicas with partial rollout",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "dep-uid-1",
					Name:      "web",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
					Strategy: appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas:     2,
					UpdatedReplicas:   1,
					AvailableReplicas: 2,
				},
			},
			wantReplicas:  3,
			wantReady:     2,
			wantUpdated:   1,
			wantAvailable: 2,
			wantStrategy:  "RollingUpdate",
		},
		{
			name: "nil replicas defaults to one",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "single", Namespace: "default"},
				Spec: appsv1.DeploymentSpec{
					Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas:     1,
					UpdatedReplicas:   1,
					AvailableReplicas: 1,
				},
			},
			wantReplicas:  1,
			wantReady:     1,
			wantUpdated:   1,
			wantAvailable: 1,
			wantStrategy:  "Recreate",
		},
	}

	c := testClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := c.deploymentToDeploymentInfo(tt.dep)

			if info.Kind != "Deployment" {
				t.Errorf("Kind = %q, want %q", info.Kind, "Deployment")
			}
			if info.Replicas != tt.wantReplicas {
				t.Errorf("Replicas = %d, want %d", info.Replicas, tt.wantReplicas)
			}
			if info.ReadyReplicas != tt.wantReady {
				t.Errorf("ReadyReplicas = %d, want %d", info.ReadyReplicas, tt.wantReady)
			}
			if info.UpdatedReplicas != tt.wantUpdated {
				t.Errorf("UpdatedReplicas = %d, want %d", info.UpdatedReplicas, tt.wantUpdated)
			}
			if info.AvailableReplicas != tt.wantAvailable {
				t.Errorf("AvailableReplicas = %d, want %d", info.AvailableReplicas, tt.wantAvailable)
			}
			if info.Strategy != tt.wantStrategy {
				t.Errorf("Strategy = %q, want %q", info.Strategy, tt.wantStrategy)
			}
		})
	}
}

func TestServiceToServiceInfo(t *testing.T) {
	tests := []struct {
		name           string
		svc            *corev1.Service
		wantType       string
		wantClusterIP  string
		wantExternalIP string
		wantPorts      []ServicePort
	}{
		{
			name: "cluster ip with named port",
			svc: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
				Spec: corev1.ServiceSpec{
					Type:      corev1.ServiceTypeClusterIP,
					ClusterIP: "10.96.0.10",
					Ports: []corev1.ServicePort{
						{
							Name:       "http",
							Port:       80,
							TargetPort: intstr.FromInt32(8080),
							Protocol:   corev1.ProtocolTCP,
						},
						{
							Name:       "metrics",
							Port:       9090,
							TargetPort: intstr.FromString("metrics"),
							Protocol:   corev1.ProtocolTCP,
						},
					},
				},
			},
			wantType:      "ClusterIP",
			wantClusterIP: "10.96.0.10",
			wantPorts: []ServicePort{
				{Name: "http", Port: 80, TargetPort: "8080", Protocol: "TCP"},
				{Name: "metrics", Port: 9090, TargetPort: "metrics", Protocol: "TCP"},
			},
		},
		{
			name: "load balancer with ingress ip",
			svc: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "lb", Namespace: "default"},
				Spec: corev1.ServiceSpec{
					Type:      corev1.ServiceTypeLoadBalancer,
					ClusterIP: "10.96.0.20",
					Ports: []corev1.ServicePort{
						{
							Name:       "https",
							Port:       443,
							TargetPort: intstr.FromInt32(8443),
							Protocol:   corev1.ProtocolTCP,
							NodePort:   30443,
						},
					},
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: "203.0.113.10"},
							{Hostname: "lb.example.com"},
						},
					},
				},
			},
			wantType:       "LoadBalancer",
			wantClusterIP:  "10.96.0.20",
			wantExternalIP: "203.0.113.10,lb.example.com",
			wantPorts: []ServicePort{
				{Name: "https", Port: 443, TargetPort: "8443", Protocol: "TCP", NodePort: 30443},
			},
		},
		{
			name: "explicit external ips take precedence",
			svc: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "ext", Namespace: "default"},
				Spec: corev1.ServiceSpec{
					Type:        corev1.ServiceTypeClusterIP,
					ClusterIP:   "10.96.0.30",
					ExternalIPs: []string{"192.0.2.1", "192.0.2.2"},
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{IP: "203.0.113.99"}},
					},
				},
			},
			wantType:       "ClusterIP",
			wantClusterIP:  "10.96.0.30",
			wantExternalIP: "192.0.2.1,192.0.2.2",
		},
	}

	c := testClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := c.serviceToServiceInfo(tt.svc)

			if info.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", info.Type, tt.wantType)
			}
			if info.ClusterIP != tt.wantClusterIP {
				t.Errorf("ClusterIP = %q, want %q", info.ClusterIP, tt.wantClusterIP)
			}
			if info.ExternalIP != tt.wantExternalIP {
				t.Errorf("ExternalIP = %q, want %q", info.ExternalIP, tt.wantExternalIP)
			}
			if tt.wantPorts != nil && !reflect.DeepEqual(info.Ports, tt.wantPorts) {
				t.Errorf("Ports = %+v, want %+v", info.Ports, tt.wantPorts)
			}
		})
	}
}

func TestNodeToNodeInfo(t *testing.T) {
	tests := []struct {
		name       string
		node       *corev1.Node
		wantStatus string
		wantRoles  []string
		wantTaints []string
	}{
		{
			name: "ready control plane node",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cp-1",
					Labels: map[string]string{
						"node-role.kubernetes.io/control-plane": "",
					},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: corev1.TaintEffectNoSchedule},
					},
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
					Addresses: []corev1.NodeAddress{
						{Type: corev1.NodeInternalIP, Address: "10.0.0.1"},
						{Type: corev1.NodeExternalIP, Address: "198.51.100.1"},
					},
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion:  "v1.31.0",
						OperatingSystem: "linux",
						Architecture:    "amd64",
					},
				},
			},
			wantStatus: "Ready",
			wantRoles:  []string{"control-plane"},
			wantTaints: []string{"node-role.kubernetes.io/control-plane:NoSchedule"},
		},
		{
			name: "not ready worker node",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
					},
				},
			},
			wantStatus: "NotReady",
			wantRoles:  []string{"<none>"},
			wantTaints: []string{"<none>"},
		},
		{
			name: "no ready condition means unknown",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "worker-2"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
					},
				},
			},
			wantStatus: "Unknown",
			wantRoles:  []string{"<none>"},
			wantTaints: []string{"<none>"},
		},
	}

	c := testClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := c.nodeToNodeInfo(tt.node)

			if info.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", info.Status, tt.wantStatus)
			}
			if !reflect.DeepEqual(info.Roles, tt.wantRoles) {
				t.Errorf("Roles = %v, want %v", info.Roles, tt.wantRoles)
			}
			if !reflect.DeepEqual(info.Taints, tt.wantTaints) {
				t.Errorf("Taints = %v, want %v", info.Taints, tt.wantTaints)
			}
		})
	}
}

func TestNodeToNodeInfoAddresses(t *testing.T) {
	c := testClient()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "10.0.0.1"},
				{Type: corev1.NodeExternalIP, Address: "198.51.100.1"},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion:  "v1.31.0",
				OperatingSystem: "linux",
				Architecture:    "arm64",
			},
		},
	}

	info := c.nodeToNodeInfo(node)

	if info.InternalIP != "10.0.0.1" {
		t.Errorf("InternalIP = %q, want %q", info.InternalIP, "10.0.0.1")
	}
	if info.ExternalIP != "198.51.100.1" {
		t.Errorf("ExternalIP = %q, want %q", info.ExternalIP, "198.51.100.1")
	}
	if info.Version != "v1.31.0" {
		t.Errorf("Version = %q, want %q", info.Version, "v1.31.0")
	}
	if info.OS != "linux" || info.Arch != "arm64" {
		t.Errorf("OS/Arch = %q/%q, want linux/arm64", info.OS, info.Arch)
	}
}
