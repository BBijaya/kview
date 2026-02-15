package k8s

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// ContextInfo contains information about a kubeconfig context
type ContextInfo struct {
	Name      string
	Cluster   string
	Namespace string
	User      string
	Current   bool
}

// GetKubeconfig returns the path to the kubeconfig file
func GetKubeconfig() string {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

// LoadKubeconfig loads the kubeconfig from the default path
func LoadKubeconfig() (*api.Config, error) {
	kubeconfig := GetKubeconfig()
	return clientcmd.LoadFromFile(kubeconfig)
}

// GetContexts returns all available contexts from kubeconfig
func GetContexts() ([]ContextInfo, error) {
	config, err := LoadKubeconfig()
	if err != nil {
		return nil, err
	}

	var contexts []ContextInfo
	for name, ctx := range config.Contexts {
		contexts = append(contexts, ContextInfo{
			Name:      name,
			Cluster:   ctx.Cluster,
			Namespace: ctx.Namespace,
			User:      ctx.AuthInfo,
			Current:   name == config.CurrentContext,
		})
	}
	return contexts, nil
}

// GetCurrentContext returns the current context name
func GetCurrentContext() (string, error) {
	config, err := LoadKubeconfig()
	if err != nil {
		return "", err
	}
	return config.CurrentContext, nil
}

// GetCurrentContextInfo returns the ContextInfo for the current context
func GetCurrentContextInfo() (*ContextInfo, error) {
	config, err := LoadKubeconfig()
	if err != nil {
		return nil, err
	}

	currentContext := config.CurrentContext
	ctx, ok := config.Contexts[currentContext]
	if !ok {
		return nil, nil
	}

	return &ContextInfo{
		Name:      currentContext,
		Cluster:   ctx.Cluster,
		Namespace: ctx.Namespace,
		User:      ctx.AuthInfo,
		Current:   true,
	}, nil
}

// GetCurrentUser returns the current user name from kubeconfig
func GetCurrentUser() string {
	info, err := GetCurrentContextInfo()
	if err != nil || info == nil {
		return ""
	}
	return info.User
}

// GetCurrentClusterName returns the current cluster name from kubeconfig
func GetCurrentClusterName() string {
	info, err := GetCurrentContextInfo()
	if err != nil || info == nil {
		return ""
	}
	return info.Cluster
}

// BuildConfigFromContext builds a REST config for a specific context
func BuildConfigFromContext(contextName string) (*rest.Config, error) {
	kubeconfig := GetKubeconfig()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	).ClientConfig()
}

// BuildDefaultConfig builds a REST config for the current context
func BuildDefaultConfig() (*rest.Config, error) {
	kubeconfig := GetKubeconfig()
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// GetAvailableContexts returns a list of context names
func GetAvailableContexts() ([]string, error) {
	config, err := LoadKubeconfig()
	if err != nil {
		return nil, err
	}

	var names []string
	for name := range config.Contexts {
		names = append(names, name)
	}
	return names, nil
}

// GetNamespaces returns all namespaces for a given context
func GetNamespaces(client Client) ([]string, error) {
	ctx := client.Context()
	resources, err := client.List(ctx, "namespaces", "")
	if err != nil {
		return nil, err
	}

	var namespaces []string
	for _, r := range resources {
		namespaces = append(namespaces, r.Name)
	}
	return namespaces, nil
}
