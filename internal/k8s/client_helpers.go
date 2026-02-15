package k8s

import (
	"context"
	"fmt"
	"io"
	"strings"

	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func (c *K8sClient) getGVR(kind string) (schema.GroupVersionResource, error) {
	kindToGVR := map[string]schema.GroupVersionResource{
		"pods":        {Group: "", Version: "v1", Resource: "pods"},
		"pod":         {Group: "", Version: "v1", Resource: "pods"},
		"services":    {Group: "", Version: "v1", Resource: "services"},
		"service":     {Group: "", Version: "v1", Resource: "services"},
		"svc":         {Group: "", Version: "v1", Resource: "services"},
		"namespaces":  {Group: "", Version: "v1", Resource: "namespaces"},
		"namespace":   {Group: "", Version: "v1", Resource: "namespaces"},
		"ns":          {Group: "", Version: "v1", Resource: "namespaces"},
		"configmaps":  {Group: "", Version: "v1", Resource: "configmaps"},
		"configmap":   {Group: "", Version: "v1", Resource: "configmaps"},
		"cm":          {Group: "", Version: "v1", Resource: "configmaps"},
		"secrets":     {Group: "", Version: "v1", Resource: "secrets"},
		"secret":      {Group: "", Version: "v1", Resource: "secrets"},
		"events":      {Group: "", Version: "v1", Resource: "events"},
		"event":       {Group: "", Version: "v1", Resource: "events"},
		"nodes":       {Group: "", Version: "v1", Resource: "nodes"},
		"node":        {Group: "", Version: "v1", Resource: "nodes"},
		"deployments": {Group: "apps", Version: "v1", Resource: "deployments"},
		"deployment":  {Group: "apps", Version: "v1", Resource: "deployments"},
		"deploy":      {Group: "apps", Version: "v1", Resource: "deployments"},
		"replicasets": {Group: "apps", Version: "v1", Resource: "replicasets"},
		"replicaset":  {Group: "apps", Version: "v1", Resource: "replicasets"},
		"rs":          {Group: "apps", Version: "v1", Resource: "replicasets"},
		"daemonsets":  {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"daemonset":   {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"ds":          {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"statefulsets": {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"statefulset":  {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"sts":          {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"ingresses":               {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"ingress":                 {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"ing":                     {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"persistentvolumeclaims":  {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"persistentvolumeclaim":   {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"pvc":                     {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"pvcs":                    {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"jobs":                    {Group: "batch", Version: "v1", Resource: "jobs"},
		"job":                     {Group: "batch", Version: "v1", Resource: "jobs"},
		"cronjobs":                {Group: "batch", Version: "v1", Resource: "cronjobs"},
		"cronjob":                 {Group: "batch", Version: "v1", Resource: "cronjobs"},
		"cj":                      {Group: "batch", Version: "v1", Resource: "cronjobs"},
		"horizontalpodautoscalers": {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
		"horizontalpodautoscaler":  {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
		"hpa":                      {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
		"persistentvolumes":        {Group: "", Version: "v1", Resource: "persistentvolumes"},
		"persistentvolume":         {Group: "", Version: "v1", Resource: "persistentvolumes"},
		"pv":                       {Group: "", Version: "v1", Resource: "persistentvolumes"},
		"rolebindings":             {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
		"rolebinding":              {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
		"rb":                       {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
	}

	gvr, ok := kindToGVR[strings.ToLower(kind)]
	if ok {
		return gvr, nil
	}

	// Fallback to discovery registry
	if c.apiRegistry != nil {
		if info, found := c.apiRegistry.Lookup(kind); found {
			return info.GVR, nil
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("unknown resource kind: %s", kind)
}

func (c *K8sClient) isClusterScoped(kind string) bool {
	clusterScoped := map[string]bool{
		"namespaces":         true,
		"namespace":          true,
		"ns":                 true,
		"nodes":              true,
		"node":               true,
		"persistentvolumes":  true,
		"persistentvolume":   true,
		"pv":                 true,
	}
	if val, ok := clusterScoped[strings.ToLower(kind)]; ok {
		return val
	}

	// Fallback to discovery registry
	if c.apiRegistry != nil {
		if info, found := c.apiRegistry.Lookup(kind); found {
			return !info.Namespaced
		}
	}

	return false
}

// CheckWriteAccess checks if the current user has write access to the cluster.
// Returns "RW" if allowed, "RO" if denied, or "RW" on error (optimistic default).
func (c *K8sClient) CheckWriteAccess(ctx context.Context) string {
	sar := &authzv1.SelfSubjectAccessReview{
		Spec: authzv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{
				Namespace: "default",
				Verb:      "create",
				Resource:  "pods",
			},
		},
	}
	result, err := c.clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return "RW" // optimistic default on error
	}
	if result.Status.Allowed {
		return "RW"
	}
	return "RO"
}

// DisconnectedClient is a placeholder client when no kubeconfig is available
type DisconnectedClient struct {
	errorMessage string
}

// NewDisconnectedClient creates a client that returns errors for all operations
func NewDisconnectedClient(errMsg string) *DisconnectedClient {
	return &DisconnectedClient{errorMessage: errMsg}
}

// ErrorMessage returns the connection error message
func (c *DisconnectedClient) ErrorMessage() string {
	return c.errorMessage
}

func (c *DisconnectedClient) List(ctx context.Context, kind, namespace string) ([]Resource, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) Get(ctx context.Context, kind, namespace, name string) (*Resource, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) Watch(ctx context.Context, kind, namespace string) (<-chan WatchEvent, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) GetClusterMetrics(ctx context.Context) (*ClusterMetrics, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListPodMetrics(ctx context.Context, namespace string) ([]PodMetrics, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListNodeMetrics(ctx context.Context) ([]NodeMetrics, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) Delete(ctx context.Context, kind, namespace, name string) error {
	return fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) Restart(ctx context.Context, kind, namespace, name string) error {
	return fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) Scale(ctx context.Context, kind, namespace, name string, replicas int) error {
	return fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) Logs(ctx context.Context, namespace, pod, container string, opts LogOptions) (io.ReadCloser, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListPods(ctx context.Context, namespace string) ([]PodInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListDeployments(ctx context.Context, namespace string) ([]DeploymentInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListServices(ctx context.Context, namespace string) ([]ServiceInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListConfigMaps(ctx context.Context, namespace string) ([]ConfigMapInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListSecrets(ctx context.Context, namespace string) ([]SecretInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) GetSecretDecoded(ctx context.Context, namespace, name string) (string, error) {
	return "", fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListIngresses(ctx context.Context, namespace string) ([]IngressInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListPVCs(ctx context.Context, namespace string) ([]PVCInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListStatefulSets(ctx context.Context, namespace string) ([]StatefulSetInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListNodes(ctx context.Context) ([]NodeInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListEvents(ctx context.Context, namespace string) ([]EventInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListReplicaSets(ctx context.Context, namespace string) ([]ReplicaSetInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListDaemonSets(ctx context.Context, namespace string) ([]DaemonSetInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListJobs(ctx context.Context, namespace string) ([]JobInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListCronJobs(ctx context.Context, namespace string) ([]CronJobInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) GetNamespaces(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListNamespaceInfos(ctx context.Context) ([]NamespaceInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) GetContexts() ([]string, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ClusterID() string {
	return ""
}

func (c *DisconnectedClient) ServerVersion() string {
	return "disconnected"
}

func (c *DisconnectedClient) Context() context.Context {
	return context.Background()
}

func (c *DisconnectedClient) CheckWriteAccess(ctx context.Context) string {
	return "n/a"
}

func (c *DisconnectedClient) GetRestConfig() *rest.Config {
	return nil
}

func (c *DisconnectedClient) GetClientset() *kubernetes.Clientset {
	return nil
}

func (c *DisconnectedClient) ListHPAs(ctx context.Context, namespace string) ([]HPAInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListPVs(ctx context.Context) ([]PVInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListRoleBindings(ctx context.Context, namespace string) ([]RoleBindingInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListHelmReleases(ctx context.Context, namespace string) ([]HelmReleaseInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) ListHelmReleaseHistory(ctx context.Context, namespace, releaseName string) ([]HelmReleaseInfo, error) {
	return nil, fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) GetHelmValues(ctx context.Context, namespace, releaseName string, revision int) (string, error) {
	return "", fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) GetHelmManifest(ctx context.Context, namespace, releaseName string, revision int) (string, error) {
	return "", fmt.Errorf(c.errorMessage)
}

func (c *DisconnectedClient) APIResources() *APIResourceRegistry {
	return NewAPIResourceRegistry()
}
