package k8s

import (
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Client defines the interface for Kubernetes operations
type Client interface {
	// Resource operations
	List(ctx context.Context, kind, namespace string) ([]Resource, error)
	Get(ctx context.Context, kind, namespace, name string) (*Resource, error)
	Watch(ctx context.Context, kind, namespace string) (<-chan WatchEvent, error)

	// Actions
	Delete(ctx context.Context, kind, namespace, name string) error
	Restart(ctx context.Context, kind, namespace, name string) error
	Scale(ctx context.Context, kind, namespace, name string, replicas int) error
	Logs(ctx context.Context, namespace, pod, container string, opts LogOptions) (io.ReadCloser, error)

	// Pod-specific
	ListPods(ctx context.Context, namespace string) ([]PodInfo, error)
	ListDeployments(ctx context.Context, namespace string) ([]DeploymentInfo, error)
	ListServices(ctx context.Context, namespace string) ([]ServiceInfo, error)

	// Additional resources
	ListConfigMaps(ctx context.Context, namespace string) ([]ConfigMapInfo, error)
	ListSecrets(ctx context.Context, namespace string) ([]SecretInfo, error)
	GetSecretDecoded(ctx context.Context, namespace, name string) (string, error)
	ListIngresses(ctx context.Context, namespace string) ([]IngressInfo, error)
	ListPVCs(ctx context.Context, namespace string) ([]PVCInfo, error)
	ListStatefulSets(ctx context.Context, namespace string) ([]StatefulSetInfo, error)

	// Phase 4: New resource types
	ListNodes(ctx context.Context) ([]NodeInfo, error)
	ListEvents(ctx context.Context, namespace string) ([]EventInfo, error)
	ListReplicaSets(ctx context.Context, namespace string) ([]ReplicaSetInfo, error)
	ListDaemonSets(ctx context.Context, namespace string) ([]DaemonSetInfo, error)
	ListJobs(ctx context.Context, namespace string) ([]JobInfo, error)
	ListCronJobs(ctx context.Context, namespace string) ([]CronJobInfo, error)

	// Phase 5: HPA, PV, RoleBindings
	ListHPAs(ctx context.Context, namespace string) ([]HPAInfo, error)
	ListPVs(ctx context.Context) ([]PVInfo, error)
	ListRoleBindings(ctx context.Context, namespace string) ([]RoleBindingInfo, error)

	// Helm releases (from Helm 3 Secrets)
	ListHelmReleases(ctx context.Context, namespace string) ([]HelmReleaseInfo, error)
	ListHelmReleaseHistory(ctx context.Context, namespace, releaseName string) ([]HelmReleaseInfo, error)
	GetHelmValues(ctx context.Context, namespace, releaseName string, revision int) (string, error)
	GetHelmManifest(ctx context.Context, namespace, releaseName string, revision int) (string, error)

	// Namespace and context operations
	GetNamespaces(ctx context.Context) ([]string, error)
	ListNamespaceInfos(ctx context.Context) ([]NamespaceInfo, error)
	GetContexts() ([]string, error)

	// Metrics
	GetClusterMetrics(ctx context.Context) (*ClusterMetrics, error)
	ListPodMetrics(ctx context.Context, namespace string) ([]PodMetrics, error)
	ListNodeMetrics(ctx context.Context) ([]NodeMetrics, error)

	// Access check
	CheckWriteAccess(ctx context.Context) string

	// Metadata
	ClusterID() string
	ServerVersion() string
	Context() context.Context

	// Discovery
	APIResources() *APIResourceRegistry

	// Client accessors (for shell exec)
	GetRestConfig() *rest.Config
	GetClientset() *kubernetes.Clientset
}

// ClusterMetrics contains aggregated cluster CPU and memory usage
type ClusterMetrics struct {
	CPUUsage    string
	CPUCapacity string
	MemUsage    string
	MemCapacity string
}

// K8sClient implements the Client interface
type K8sClient struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	config        *rest.Config
	clusterID     string
	serverVersion string
	ctx           context.Context
	apiRegistry   *APIResourceRegistry
}

// NewClient creates a new Kubernetes client for the current context
func NewClient() (*K8sClient, error) {
	config, err := BuildDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}
	return NewClientWithConfig(config)
}

// NewClientForContext creates a new Kubernetes client for a specific context
func NewClientForContext(contextName string) (*K8sClient, error) {
	config, err := BuildConfigFromContext(contextName)
	if err != nil {
		return nil, fmt.Errorf("failed to build config for context %s: %w", contextName, err)
	}
	return NewClientWithConfig(config)
}

// NewClientWithConfig creates a new Kubernetes client with a specific config
func NewClientWithConfig(config *rest.Config) (*K8sClient, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Get server version
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}

	// Generate cluster ID from server URL
	clusterID := config.Host

	client := &K8sClient{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		config:        config,
		clusterID:     clusterID,
		serverVersion: version.GitVersion,
		ctx:           context.Background(),
		apiRegistry:   NewAPIResourceRegistry(),
	}

	// Discover API resources in background (non-blocking)
	go client.apiRegistry.Discover(clientset)

	return client, nil
}

func (c *K8sClient) ClusterID() string {
	return c.clusterID
}

func (c *K8sClient) ServerVersion() string {
	return c.serverVersion
}

func (c *K8sClient) Context() context.Context {
	return c.ctx
}

func (c *K8sClient) GetRestConfig() *rest.Config {
	return c.config
}

func (c *K8sClient) GetClientset() *kubernetes.Clientset {
	return c.clientset
}

func (c *K8sClient) APIResources() *APIResourceRegistry {
	return c.apiRegistry
}

// List lists resources of a given kind in a namespace
func (c *K8sClient) List(ctx context.Context, kind, namespace string) ([]Resource, error) {
	gvr, err := c.getGVR(kind)
	if err != nil {
		return nil, err
	}

	var list *unstructured.UnstructuredList
	if namespace == "" || c.isClusterScoped(kind) {
		list, err = c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		list, err = c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var resources []Resource
	for _, item := range list.Items {
		resources = append(resources, c.unstructuredToResource(&item))
	}
	return resources, nil
}

// Get gets a specific resource
func (c *K8sClient) Get(ctx context.Context, kind, namespace, name string) (*Resource, error) {
	gvr, err := c.getGVR(kind)
	if err != nil {
		return nil, err
	}

	var obj *unstructured.Unstructured
	if namespace == "" || c.isClusterScoped(kind) {
		obj, err = c.dynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	} else {
		obj, err = c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	resource := c.unstructuredToResource(obj)
	return &resource, nil
}

// Watch watches resources of a given kind
func (c *K8sClient) Watch(ctx context.Context, kind, namespace string) (<-chan WatchEvent, error) {
	gvr, err := c.getGVR(kind)
	if err != nil {
		return nil, err
	}

	var watcher watch.Interface
	if namespace == "" || c.isClusterScoped(kind) {
		watcher, err = c.dynamicClient.Resource(gvr).Watch(ctx, metav1.ListOptions{})
	} else {
		watcher, err = c.dynamicClient.Resource(gvr).Namespace(namespace).Watch(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	events := make(chan WatchEvent)
	go func() {
		defer close(events)
		for event := range watcher.ResultChan() {
			if obj, ok := event.Object.(*unstructured.Unstructured); ok {
				resource := c.unstructuredToResource(obj)
				events <- WatchEvent{
					Type:     WatchEventType(event.Type),
					Resource: &resource,
				}
			}
		}
	}()

	return events, nil
}

// Delete deletes a resource
func (c *K8sClient) Delete(ctx context.Context, kind, namespace, name string) error {
	gvr, err := c.getGVR(kind)
	if err != nil {
		return err
	}

	if namespace == "" || c.isClusterScoped(kind) {
		return c.dynamicClient.Resource(gvr).Delete(ctx, name, metav1.DeleteOptions{})
	}
	return c.dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// Restart restarts a deployment by updating its annotations
func (c *K8sClient) Restart(ctx context.Context, kind, namespace, name string) error {
	if kind != "deployments" && kind != "deployment" {
		return fmt.Errorf("restart is only supported for deployments")
	}

	deployment, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = c.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

// Scale scales a deployment or statefulset
func (c *K8sClient) Scale(ctx context.Context, kind, namespace, name string, replicas int) error {
	switch kind {
	case "deployments", "deployment":
		scale, err := c.clientset.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		scale.Spec.Replicas = int32(replicas)
		_, err = c.clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
		return err
	case "statefulsets", "statefulset":
		scale, err := c.clientset.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		scale.Spec.Replicas = int32(replicas)
		_, err = c.clientset.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("scale is only supported for deployments and statefulsets")
	}
}

// Logs returns logs for a pod
func (c *K8sClient) Logs(ctx context.Context, namespace, pod, container string, opts LogOptions) (io.ReadCloser, error) {
	podLogOpts := &corev1.PodLogOptions{
		Container: container,
		Follow:    opts.Follow,
		Previous:  opts.Previous,
	}
	if opts.TailLines > 0 {
		podLogOpts.TailLines = &opts.TailLines
	}
	if opts.SinceSeconds > 0 {
		podLogOpts.SinceSeconds = &opts.SinceSeconds
	}
	if opts.Timestamps {
		podLogOpts.Timestamps = true
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(pod, podLogOpts)
	return req.Stream(ctx)
}
