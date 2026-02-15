package views

import (
	"context"
	"fmt"
	"sort"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bijaya/kview/internal/k8s"
)

// Refresh fetches all cluster data for the health dashboard
func (v *HealthView) Refresh() tea.Cmd {
	v.loading = true
	return func() tea.Msg {
		ctx := context.Background()
		result := HealthDataMsg{}

		if metrics, err := v.client.GetClusterMetrics(ctx); err == nil {
			result.ClusterMetrics = metrics
		}

		pods, err := v.client.ListPods(ctx, v.namespace)
		if err != nil {
			result.Err = fmt.Errorf("failed to list pods: %w", err)
			return result
		}
		result.Pods = pods

		if nodes, err := v.client.ListNodes(ctx); err == nil {
			result.Nodes = nodes
		}

		if nm, err := v.client.ListNodeMetrics(ctx); err == nil {
			result.NodeMetrics = nm
		}

		if d, err := v.client.ListDeployments(ctx, v.namespace); err == nil {
			result.Deployments = d
		}

		if s, err := v.client.ListStatefulSets(ctx, v.namespace); err == nil {
			result.StatefulSets = s
		}

		if d, err := v.client.ListDaemonSets(ctx, v.namespace); err == nil {
			result.DaemonSets = d
		}

		if j, err := v.client.ListJobs(ctx, v.namespace); err == nil {
			result.Jobs = j
		}

		if pvcs, err := v.client.ListPVCs(ctx, v.namespace); err == nil {
			result.PVCs = pvcs
		}

		result.Diagnoses = v.ruleSet.Analyze(nil, pods, nil)

		if e, err := v.client.ListEvents(ctx, v.namespace); err == nil {
			result.Events = e
		}

		return result
	}
}

// --- Sort helpers ---

func (v *HealthView) sortDiagnoses() {
	sort.SliceStable(v.diagnoses, func(i, j int) bool {
		return severityRank(v.diagnoses[i].Severity) < severityRank(v.diagnoses[j].Severity)
	})
}

func (v *HealthView) sortEvents() {
	sort.SliceStable(v.events, func(i, j int) bool {
		if v.events[i].Type != v.events[j].Type {
			if v.events[i].Type == "Warning" {
				return true
			}
			if v.events[j].Type == "Warning" {
				return false
			}
		}
		return v.events[i].LastSeen.After(v.events[j].LastSeen)
	})
}

// --- Build display lists ---

func (v *HealthView) buildDisplayedNodes() {
	nodeMetricsMap := make(map[string]k8s.NodeMetrics)
	for _, nm := range v.nodeMetrics {
		nodeMetricsMap[nm.Name] = nm
	}

	var notReady, ready []healthNodeEntry
	for _, n := range v.nodes {
		entry := healthNodeEntry{node: n, cpuPct: -1, memPct: -1}
		if nm, ok := nodeMetricsMap[n.Name]; ok {
			if n.CPUAllocatable > 0 {
				entry.cpuPct = int(nm.CPUUsage * 100 / n.CPUAllocatable)
			}
			if n.MemAllocatable > 0 {
				entry.memPct = int(nm.MemUsage * 100 / n.MemAllocatable)
			}
		}
		if n.Status != "Ready" {
			notReady = append(notReady, entry)
		} else {
			ready = append(ready, entry)
		}
	}

	// Sort ready nodes by CPU% descending (hot nodes first)
	sort.SliceStable(ready, func(i, j int) bool {
		return ready[i].cpuPct > ready[j].cpuPct
	})

	// All NotReady + top 3 Ready
	v.displayedNodes = nil
	v.displayedNodes = append(v.displayedNodes, notReady...)
	maxReady := min(3, len(ready))
	v.displayedNodes = append(v.displayedNodes, ready[:maxReady]...)
}

func (v *HealthView) buildUnhealthyWorkloads() {
	v.unhealthyWorkloads = nil
	for _, d := range v.deployments {
		if d.ReadyReplicas < d.Replicas {
			v.unhealthyWorkloads = append(v.unhealthyWorkloads, unhealthyWorkload{
				Kind: "Deployment", Resource: "deployments",
				Name: d.Name, Namespace: d.Namespace, UID: d.UID,
				Ready: d.ReadyReplicas, Desired: d.Replicas, Age: d.Age,
			})
		}
	}
	for _, s := range v.statefulsets {
		if s.ReadyReplicas < s.Replicas {
			v.unhealthyWorkloads = append(v.unhealthyWorkloads, unhealthyWorkload{
				Kind: "StatefulSet", Resource: "statefulsets",
				Name: s.Name, Namespace: s.Namespace, UID: s.UID,
				Ready: s.ReadyReplicas, Desired: s.Replicas, Age: s.Age,
			})
		}
	}
	for _, d := range v.daemonsets {
		if d.ReadyNumber < d.DesiredNumber {
			v.unhealthyWorkloads = append(v.unhealthyWorkloads, unhealthyWorkload{
				Kind: "DaemonSet", Resource: "daemonsets",
				Name: d.Name, Namespace: d.Namespace, UID: d.UID,
				Ready: d.ReadyNumber, Desired: d.DesiredNumber, Age: d.Age,
			})
		}
	}
	// Sort by gap (desired - ready) descending — worst first
	sort.SliceStable(v.unhealthyWorkloads, func(i, j int) bool {
		gapI := v.unhealthyWorkloads[i].Desired - v.unhealthyWorkloads[i].Ready
		gapJ := v.unhealthyWorkloads[j].Desired - v.unhealthyWorkloads[j].Ready
		return gapI > gapJ
	})
}

func (v *HealthView) buildFailedJobs() {
	v.failedJobs = nil
	for _, j := range v.jobs {
		if j.Failed > 0 || j.Status == "Failed" {
			v.failedJobs = append(v.failedJobs, j)
		}
	}
	sort.SliceStable(v.failedJobs, func(i, j int) bool {
		return v.failedJobs[i].Failed > v.failedJobs[j].Failed
	})
	if len(v.failedJobs) > 10 {
		v.failedJobs = v.failedJobs[:10]
	}
}

func (v *HealthView) buildProblemPods() {
	v.problemPods = nil
	for _, p := range v.pods {
		switch p.Phase {
		case "Running", "Succeeded":
			continue
		}
		v.problemPods = append(v.problemPods, p)
	}
	// Sort: Failed first, then Pending, then others; within same phase by age descending (longest stuck first)
	sort.SliceStable(v.problemPods, func(i, j int) bool {
		ri := problemPhaseRank(v.problemPods[i].Phase)
		rj := problemPhaseRank(v.problemPods[j].Phase)
		if ri != rj {
			return ri < rj
		}
		return v.problemPods[i].Age > v.problemPods[j].Age
	})
	if len(v.problemPods) > 15 {
		v.problemPods = v.problemPods[:15]
	}
}

func (v *HealthView) buildPendingPVCs() {
	v.pendingPVCs = nil
	for _, pvc := range v.pvcs {
		if pvc.Status == "Pending" {
			v.pendingPVCs = append(v.pendingPVCs, pvc)
		}
	}
	// Longest pending first
	sort.SliceStable(v.pendingPVCs, func(i, j int) bool {
		return v.pendingPVCs[i].Age > v.pendingPVCs[j].Age
	})
}

func (v *HealthView) buildRestartingPods() {
	v.restartingPods = nil
	for _, p := range v.pods {
		if p.Restarts > 0 {
			v.restartingPods = append(v.restartingPods, p)
		}
	}
	sort.SliceStable(v.restartingPods, func(i, j int) bool {
		return v.restartingPods[i].Restarts > v.restartingPods[j].Restarts
	})
	if len(v.restartingPods) > 10 {
		v.restartingPods = v.restartingPods[:10]
	}
}
