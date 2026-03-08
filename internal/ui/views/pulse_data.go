package views

import (
	"context"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/theme"
)

// gaugeCount is the number of resource gauges in the pulse view.
const gaugeCount = 16

// Gauge index constants for the 16 resource types.
const (
	gaugePods = iota
	gaugeDeployments
	gaugeReplicaSets
	gaugeDaemonSets
	gaugeStatefulSets
	gaugeJobs
	gaugeCronJobs
	gaugeServices
	gaugeIngresses
	gaugeConfigMaps
	gaugeSecrets
	gaugePVCs
	gaugePVs
	gaugeHPAs
	gaugeNodes
	gaugeEvents
)

// gaugeNames maps gauge index to display name.
var gaugeNames = [gaugeCount]string{
	"Pods", "Deploy", "RS", "DS", "STS",
	"Jobs", "CronJobs", "Svc", "Ing",
	"CM", "Sec", "PVC", "PV", "HPA",
	"Nodes", "Events",
}

// gaugeViewTypes maps gauge index to the corresponding list ViewType for drill-down.
var gaugeViewTypes = [gaugeCount]theme.ViewType{
	theme.ViewPods, theme.ViewDeployments, theme.ViewReplicaSets,
	theme.ViewDaemonSets, theme.ViewStatefulSets, theme.ViewJobs,
	theme.ViewCronJobs, theme.ViewServices, theme.ViewIngresses,
	theme.ViewConfigMaps, theme.ViewSecrets, theme.ViewPVCs,
	theme.ViewPVs, theme.ViewHPAs, theme.ViewNodes, theme.ViewEvents,
}

// GaugeData holds OK/Fault counts for a single resource type.
type GaugeData struct {
	Name     string
	OK       int
	Fault    int
	ViewType theme.ViewType
}

// PulseDataMsg is sent when pulse data has been fetched.
type PulseDataMsg struct {
	Gauges [gaugeCount]GaugeData
	CPUPct int
	MemPct int
	Err    error
}

// PulseTickMsg triggers periodic refresh of pulse data.
type PulseTickMsg struct{}

// pulseRefresh fetches all 16 resource types concurrently and classifies results.
func pulseRefresh(client k8s.Client, namespace string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var mu sync.Mutex
		var wg sync.WaitGroup
		var result PulseDataMsg

		// Initialize gauge names and view types
		for i := 0; i < gaugeCount; i++ {
			result.Gauges[i].Name = gaugeNames[i]
			result.Gauges[i].ViewType = gaugeViewTypes[i]
		}

		// Fetch metrics
		wg.Add(1)
		go func() {
			defer wg.Done()
			if metrics, err := client.GetClusterMetrics(ctx); err == nil && metrics != nil {
				mu.Lock()
				result.CPUPct = parsePercentage(metrics.CPUUsage)
				result.MemPct = parsePercentage(metrics.MemUsage)
				mu.Unlock()
			}
		}()

		// Pods
		wg.Add(1)
		go func() {
			defer wg.Done()
			pods, err := client.ListPods(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, p := range pods {
				switch p.Phase {
				case "Running", "Succeeded":
					ok++
				default:
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugePods].OK = ok
			result.Gauges[gaugePods].Fault = fault
			mu.Unlock()
		}()

		// Deployments
		wg.Add(1)
		go func() {
			defer wg.Done()
			deps, err := client.ListDeployments(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, d := range deps {
				if d.ReadyReplicas >= d.Replicas && d.Replicas > 0 {
					ok++
				} else {
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugeDeployments].OK = ok
			result.Gauges[gaugeDeployments].Fault = fault
			mu.Unlock()
		}()

		// ReplicaSets
		wg.Add(1)
		go func() {
			defer wg.Done()
			rsList, err := client.ListReplicaSets(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, rs := range rsList {
				if rs.DesiredReplicas == 0 {
					continue // skip scaled-down RS
				}
				if rs.ReadyReplicas >= rs.DesiredReplicas {
					ok++
				} else {
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugeReplicaSets].OK = ok
			result.Gauges[gaugeReplicaSets].Fault = fault
			mu.Unlock()
		}()

		// DaemonSets
		wg.Add(1)
		go func() {
			defer wg.Done()
			dsList, err := client.ListDaemonSets(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, ds := range dsList {
				if ds.ReadyNumber >= ds.DesiredNumber {
					ok++
				} else {
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugeDaemonSets].OK = ok
			result.Gauges[gaugeDaemonSets].Fault = fault
			mu.Unlock()
		}()

		// StatefulSets
		wg.Add(1)
		go func() {
			defer wg.Done()
			stsList, err := client.ListStatefulSets(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, sts := range stsList {
				if sts.ReadyReplicas >= sts.Replicas && sts.Replicas > 0 {
					ok++
				} else {
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugeStatefulSets].OK = ok
			result.Gauges[gaugeStatefulSets].Fault = fault
			mu.Unlock()
		}()

		// Jobs
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobList, err := client.ListJobs(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, j := range jobList {
				switch j.Status {
				case "Complete", "Completed":
					ok++
				case "Failed":
					fault++
				default:
					if j.Failed > 0 {
						fault++
					} else {
						ok++
					}
				}
			}
			mu.Lock()
			result.Gauges[gaugeJobs].OK = ok
			result.Gauges[gaugeJobs].Fault = fault
			mu.Unlock()
		}()

		// CronJobs
		wg.Add(1)
		go func() {
			defer wg.Done()
			cjList, err := client.ListCronJobs(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, cj := range cjList {
				if !cj.Suspend {
					ok++
				} else {
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugeCronJobs].OK = ok
			result.Gauges[gaugeCronJobs].Fault = fault
			mu.Unlock()
		}()

		// Services
		wg.Add(1)
		go func() {
			defer wg.Done()
			svcList, err := client.ListServices(ctx, namespace)
			if err != nil {
				return
			}
			mu.Lock()
			result.Gauges[gaugeServices].OK = len(svcList)
			result.Gauges[gaugeServices].Fault = 0
			mu.Unlock()
		}()

		// Ingresses
		wg.Add(1)
		go func() {
			defer wg.Done()
			ingList, err := client.ListIngresses(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, ing := range ingList {
				if ing.Address != "" {
					ok++
				} else {
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugeIngresses].OK = ok
			result.Gauges[gaugeIngresses].Fault = fault
			mu.Unlock()
		}()

		// ConfigMaps
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmList, err := client.ListConfigMaps(ctx, namespace)
			if err != nil {
				return
			}
			mu.Lock()
			result.Gauges[gaugeConfigMaps].OK = len(cmList)
			result.Gauges[gaugeConfigMaps].Fault = 0
			mu.Unlock()
		}()

		// Secrets
		wg.Add(1)
		go func() {
			defer wg.Done()
			secList, err := client.ListSecrets(ctx, namespace)
			if err != nil {
				return
			}
			mu.Lock()
			result.Gauges[gaugeSecrets].OK = len(secList)
			result.Gauges[gaugeSecrets].Fault = 0
			mu.Unlock()
		}()

		// PVCs
		wg.Add(1)
		go func() {
			defer wg.Done()
			pvcList, err := client.ListPVCs(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, pvc := range pvcList {
				if pvc.Status == "Bound" {
					ok++
				} else {
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugePVCs].OK = ok
			result.Gauges[gaugePVCs].Fault = fault
			mu.Unlock()
		}()

		// PVs
		wg.Add(1)
		go func() {
			defer wg.Done()
			pvList, err := client.ListPVs(ctx)
			if err != nil {
				return
			}
			var ok, fault int
			for _, pv := range pvList {
				switch pv.Status {
				case "Available", "Bound":
					ok++
				case "Released", "Failed":
					fault++
				default:
					ok++
				}
			}
			mu.Lock()
			result.Gauges[gaugePVs].OK = ok
			result.Gauges[gaugePVs].Fault = fault
			mu.Unlock()
		}()

		// HPAs
		wg.Add(1)
		go func() {
			defer wg.Done()
			hpaList, err := client.ListHPAs(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, hpa := range hpaList {
				if hpa.CurrentReplicas > 0 {
					ok++
				} else {
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugeHPAs].OK = ok
			result.Gauges[gaugeHPAs].Fault = fault
			mu.Unlock()
		}()

		// Nodes
		wg.Add(1)
		go func() {
			defer wg.Done()
			nodeList, err := client.ListNodes(ctx)
			if err != nil {
				return
			}
			var ok, fault int
			for _, n := range nodeList {
				if n.Status == "Ready" {
					ok++
				} else {
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugeNodes].OK = ok
			result.Gauges[gaugeNodes].Fault = fault
			mu.Unlock()
		}()

		// Events
		wg.Add(1)
		go func() {
			defer wg.Done()
			eventList, err := client.ListEvents(ctx, namespace)
			if err != nil {
				return
			}
			var ok, fault int
			for _, e := range eventList {
				if e.Type == "Normal" {
					ok++
				} else {
					fault++
				}
			}
			mu.Lock()
			result.Gauges[gaugeEvents].OK = ok
			result.Gauges[gaugeEvents].Fault = fault
			mu.Unlock()
		}()

		// Namespaces - not included as a gauge (16 gauges already)

		wg.Wait()
		return result
	}
}
