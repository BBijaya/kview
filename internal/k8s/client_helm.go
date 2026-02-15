package k8s

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"gopkg.in/yaml.v3"
)

// ListHelmReleases lists Helm 3 releases by querying Secrets with owner=helm label.
// Deduplicates by release name, keeping only the latest revision per release.
func (c *K8sClient) ListHelmReleases(ctx context.Context, namespace string) ([]HelmReleaseInfo, error) {
	items, err := c.listHelmSecrets(ctx, namespace, "owner=helm")
	if err != nil {
		return nil, err
	}

	// Group by release name, keep only the latest revision
	latestByName := make(map[string]*HelmReleaseInfo)
	now := time.Now()

	for _, item := range items {
		info := secretItemToHelmRelease(item, now)
		if info == nil {
			continue
		}

		existing, exists := latestByName[info.Name]
		if !exists || info.Revision > existing.Revision {
			latestByName[info.Name] = info
		}
	}

	releases := make([]HelmReleaseInfo, 0, len(latestByName))
	for _, info := range latestByName {
		releases = append(releases, *info)
	}

	return releases, nil
}

// ListHelmReleaseHistory lists all revisions for a specific Helm release.
// Returns all revisions sorted by revision number (descending).
func (c *K8sClient) ListHelmReleaseHistory(ctx context.Context, namespace, releaseName string) ([]HelmReleaseInfo, error) {
	selector := "owner=helm,name=" + releaseName
	items, err := c.listHelmSecrets(ctx, namespace, selector)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	releases := make([]HelmReleaseInfo, 0, len(items))
	for _, item := range items {
		info := secretItemToHelmRelease(item, now)
		if info != nil {
			releases = append(releases, *info)
		}
	}

	// Sort by revision descending (latest first)
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].Revision > releases[j].Revision
	})

	return releases, nil
}

// listHelmSecrets lists Secrets matching the given label selector.
func (c *K8sClient) listHelmSecrets(ctx context.Context, namespace, labelSelector string) ([]secretItem, error) {
	opts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	var items []secretItem
	if namespace == "" {
		list, err := c.clientset.CoreV1().Secrets("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for i := range list.Items {
			items = append(items, secretItem{
				name:      list.Items[i].Name,
				namespace: list.Items[i].Namespace,
				uid:       string(list.Items[i].UID),
				labels:    list.Items[i].Labels,
				data:      list.Items[i].Data,
				created:   list.Items[i].CreationTimestamp.Time,
			})
		}
	} else {
		list, err := c.clientset.CoreV1().Secrets(namespace).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for i := range list.Items {
			items = append(items, secretItem{
				name:      list.Items[i].Name,
				namespace: list.Items[i].Namespace,
				uid:       string(list.Items[i].UID),
				labels:    list.Items[i].Labels,
				data:      list.Items[i].Data,
				created:   list.Items[i].CreationTimestamp.Time,
			})
		}
	}

	return items, nil
}

// secretItemToHelmRelease converts a secretItem to a HelmReleaseInfo.
// Returns nil if the Secret doesn't have a valid release name label.
func secretItemToHelmRelease(item secretItem, now time.Time) *HelmReleaseInfo {
	releaseName := item.labels["name"]
	if releaseName == "" {
		return nil
	}

	status := item.labels["status"]
	revisionStr := item.labels["version"]
	revision, _ := strconv.Atoi(revisionStr)

	info := &HelmReleaseInfo{
		Resource: Resource{
			UID:       item.uid,
			Kind:      "Secret",
			Namespace: item.namespace,
			Name:      releaseName,
			Labels:    item.labels,
		},
		Status:   status,
		Revision: revision,
		Age:      now.Sub(item.created),
	}

	// Extract chart info from release data (base64 -> gzip -> JSON)
	if releaseData, ok := item.data["release"]; ok {
		chart, chartVersion, appVersion := extractChartInfo(releaseData)
		info.Chart = chart
		info.ChartVersion = chartVersion
		info.AppVersion = appVersion
	}

	return info
}

// secretItem is a lightweight struct to hold the fields we need from a Secret.
type secretItem struct {
	name      string
	namespace string
	uid       string
	labels    map[string]string
	data      map[string][]byte
	created   time.Time
}

// GetHelmValues returns the user-supplied values for a Helm release as YAML.
func (c *K8sClient) GetHelmValues(ctx context.Context, namespace, releaseName string, revision int) (string, error) {
	secretName := fmt.Sprintf("sh.helm.release.v1.%s.v%d", releaseName, revision)
	secret, err := c.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get release secret: %w", err)
	}

	releaseData, ok := secret.Data["release"]
	if !ok {
		return "", fmt.Errorf("release secret has no release data")
	}

	parsed, err := decodeHelmRelease(releaseData)
	if err != nil {
		return "", err
	}

	config, _ := parsed["config"]
	if config == nil {
		return "# No user-supplied values\n", nil
	}

	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal values: %w", err)
	}

	return string(yamlBytes), nil
}

// GetHelmManifest returns the rendered Kubernetes manifest for a Helm release.
func (c *K8sClient) GetHelmManifest(ctx context.Context, namespace, releaseName string, revision int) (string, error) {
	secretName := fmt.Sprintf("sh.helm.release.v1.%s.v%d", releaseName, revision)
	secret, err := c.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get release secret: %w", err)
	}

	releaseData, ok := secret.Data["release"]
	if !ok {
		return "", fmt.Errorf("release secret has no release data")
	}

	parsed, err := decodeHelmRelease(releaseData)
	if err != nil {
		return "", err
	}

	manifest, _ := parsed["manifest"].(string)
	if manifest == "" {
		return "# No manifest data\n", nil
	}

	return manifest, nil
}

// decodeHelmRelease decodes the Helm release data (base64 -> gzip -> JSON)
// and returns the full parsed JSON as a map.
func decodeHelmRelease(data []byte) (map[string]interface{}, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	reader, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}
	defer reader.Close()

	jsonData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse release JSON: %w", err)
	}

	return result, nil
}

// extractChartInfo decodes the Helm release data (base64 -> gzip -> JSON)
// and extracts chart name, chart version, and app version.
func extractChartInfo(data []byte) (chart, chartVersion, appVersion string) {
	// Helm 3 stores release data as: base64(gzip(json))
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return "", "", ""
	}

	reader, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return "", "", ""
	}
	defer reader.Close()

	jsonData, err := io.ReadAll(reader)
	if err != nil {
		return "", "", ""
	}

	// Parse just enough of the JSON to get chart metadata
	var release struct {
		Chart struct {
			Metadata struct {
				Name       string `json:"name"`
				Version    string `json:"version"`
				AppVersion string `json:"appVersion"`
			} `json:"metadata"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(jsonData, &release); err != nil {
		return "", "", ""
	}

	return release.Chart.Metadata.Name, release.Chart.Metadata.Version, release.Chart.Metadata.AppVersion
}
