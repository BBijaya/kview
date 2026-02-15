package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
	"github.com/bijaya/kview/internal/ui/theme"
)

// updateContent assembles all sections into the viewport
func (v *HealthView) updateContent() {
	w := v.viewport.Width
	if w < 40 {
		w = 40
	}

	var b strings.Builder

	// Section 0: Cluster Overview
	v.sectionLineOffsets[sectionOverview] = 0
	b.WriteString(v.renderClusterOverview(w))
	b.WriteString("\n")

	// Section 1: Nodes
	v.sectionLineOffsets[sectionNodes] = strings.Count(b.String(), "\n")
	b.WriteString(v.renderNodes(w))
	b.WriteString("\n")

	// Section 2: Unhealthy Workloads
	v.sectionLineOffsets[sectionUnhealthy] = strings.Count(b.String(), "\n")
	b.WriteString(v.renderUnhealthyWorkloads(w))
	b.WriteString("\n")

	// Section 3: Failed Jobs
	v.sectionLineOffsets[sectionFailedJobs] = strings.Count(b.String(), "\n")
	b.WriteString(v.renderFailedJobs(w))
	b.WriteString("\n")

	// Section 4: Problem Pods
	v.sectionLineOffsets[sectionProblems] = strings.Count(b.String(), "\n")
	b.WriteString(v.renderProblemPods(w))
	b.WriteString("\n")

	// Section 5: Pending PVCs
	v.sectionLineOffsets[sectionPendingPVCs] = strings.Count(b.String(), "\n")
	b.WriteString(v.renderPendingPVCs(w))
	b.WriteString("\n")

	// Section 6: Restarts
	v.sectionLineOffsets[sectionRestarts] = strings.Count(b.String(), "\n")
	b.WriteString(v.renderRestarts(w))
	b.WriteString("\n")

	// Section 7: Issues
	v.sectionLineOffsets[sectionIssues] = strings.Count(b.String(), "\n")
	b.WriteString(v.renderIssues(w))
	b.WriteString("\n")

	// Section 8: Events
	v.sectionLineOffsets[sectionEvents] = strings.Count(b.String(), "\n")
	b.WriteString(v.renderEvents(w))

	v.viewport.SetContent(b.String())
}

// --- Cluster Overview ---

func (v *HealthView) renderClusterOverview(width int) string {
	var b strings.Builder

	b.WriteString(renderSectionHeader("Cluster Overview", width, v.sectionFocus == sectionOverview, -1))
	b.WriteString("\n")

	// Health summary banner
	b.WriteString(v.renderHealthSummary(width))
	b.WriteString("\n")
	b.WriteString(theme.PadToWidth("", width, theme.ColorBackground))
	b.WriteString("\n")

	leftWidth := width/2 - 1

	cpuLeft := v.renderCPULine()
	podRight := v.renderPodCountLine()
	b.WriteString(joinColumns(cpuLeft, podRight, leftWidth, width))
	b.WriteString("\n")

	memLeft := v.renderMEMLine()
	workloadRight := v.renderWorkloadCountLine()
	b.WriteString(joinColumns(memLeft, workloadRight, leftWidth, width))
	b.WriteString("\n")
	b.WriteString(theme.PadToWidth("", width, theme.ColorBackground))
	b.WriteString("\n")

	// Node capacity pressure
	if pressure := v.renderNodePressure(); pressure != "" {
		b.WriteString(theme.PadToWidth(pressure, width, theme.ColorBackground))
		b.WriteString("\n")
	}

	return b.String()
}

func (v *HealthView) renderHealthSummary(width int) string {
	var alerts []string

	// Count NotReady nodes
	var notReadyNodes int
	for _, n := range v.nodes {
		if n.Status != "Ready" {
			notReadyNodes++
		}
	}
	if notReadyNodes > 0 {
		alerts = append(alerts, theme.Styles.StatusError.Render(
			fmt.Sprintf("%s %d/%d nodes NotReady", theme.IconError, notReadyNodes, len(v.nodes))))
	}

	if n := len(v.unhealthyWorkloads); n > 0 {
		alerts = append(alerts, theme.Styles.StatusWarning.Render(
			fmt.Sprintf("%s %d unhealthy workload%s", theme.IconWarning, n, plural(n))))
	}

	if n := len(v.failedJobs); n > 0 {
		alerts = append(alerts, theme.Styles.StatusWarning.Render(
			fmt.Sprintf("%s %d failed job%s", theme.IconWarning, n, plural(n))))
	}

	if n := len(v.problemPods); n > 0 {
		alerts = append(alerts, theme.Styles.StatusWarning.Render(
			fmt.Sprintf("%s %d problem pod%s", theme.IconWarning, n, plural(n))))
	}

	if n := len(v.pendingPVCs); n > 0 {
		alerts = append(alerts, theme.Styles.StatusWarning.Render(
			fmt.Sprintf("%s %d pending PVC%s", theme.IconWarning, n, plural(n))))
	}

	// Count OOM restarts specifically
	var oomCount int
	for _, p := range v.restartingPods {
		for _, c := range p.Containers {
			if c.StateReason == "OOMKilled" && c.RestartCount > 0 {
				oomCount++
				break
			}
		}
	}
	if oomCount > 0 {
		alerts = append(alerts, theme.Styles.StatusError.Render(
			fmt.Sprintf("%s %d OOM restart%s", theme.IconError, oomCount, plural(oomCount))))
	} else if n := len(v.restartingPods); n > 0 {
		alerts = append(alerts, theme.Styles.StatusWarning.Render(
			fmt.Sprintf("%s %d restarting pod%s", theme.IconWarning, n, plural(n))))
	}

	bgStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorBackground)

	if len(alerts) == 0 {
		line := "  " + theme.Styles.StatusHealthy.Render(theme.IconSuccess+" Cluster healthy")
		return theme.PadToWidth(line, width, theme.ColorBackground)
	}

	line := "  " + bgStyle.Render(strings.Join(alerts, bgStyle.Render(" • ")))
	return theme.PadToWidth(line, width, theme.ColorBackground)
}

func (v *HealthView) renderNodePressure() string {
	nodeMetricsMap := make(map[string]k8s.NodeMetrics)
	for _, nm := range v.nodeMetrics {
		nodeMetricsMap[nm.Name] = nm
	}
	var cpuHigh, memHigh int
	for _, n := range v.nodes {
		if nm, ok := nodeMetricsMap[n.Name]; ok {
			if n.CPUAllocatable > 0 && int(nm.CPUUsage*100/n.CPUAllocatable) >= 85 {
				cpuHigh++
			}
			if n.MemAllocatable > 0 && int(nm.MemUsage*100/n.MemAllocatable) >= 85 {
				memHigh++
			}
		}
	}

	if cpuHigh == 0 && memHigh == 0 {
		return ""
	}

	var parts []string
	if cpuHigh > 0 {
		parts = append(parts, theme.Styles.StatusWarning.Render(
			fmt.Sprintf("%s %d node >85%% CPU", theme.IconWarning, cpuHigh)))
	}
	if memHigh > 0 {
		parts = append(parts, theme.Styles.StatusWarning.Render(
			fmt.Sprintf("%s %d node >85%% MEM", theme.IconWarning, memHigh)))
	}

	bgStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorBackground)
	return "  " + bgStyle.Render(strings.Join(parts, bgStyle.Render(" • ")))
}

func (v *HealthView) renderCPULine() string {
	baseStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorBackground)

	if v.clusterMetrics == nil {
		return baseStyle.Render("  CPU ") +
			lipgloss.NewStyle().Foreground(theme.ColorNAValue).Background(theme.ColorBackground).Render("n/a")
	}

	cpuPct := parsePercentage(v.clusterMetrics.CPUUsage)
	bar := renderBar(cpuPct, 20)
	label := v.clusterMetrics.CPUUsage
	if v.clusterMetrics.CPUCapacity != "" {
		label += "  (" + v.clusterMetrics.CPUCapacity + ")"
	}
	return baseStyle.Render("  CPU ") + bar + baseStyle.Render(" "+label)
}

func (v *HealthView) renderMEMLine() string {
	baseStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorBackground)

	if v.clusterMetrics == nil {
		return baseStyle.Render("  MEM ") +
			lipgloss.NewStyle().Foreground(theme.ColorNAValue).Background(theme.ColorBackground).Render("n/a")
	}

	memPct := parsePercentage(v.clusterMetrics.MemUsage)
	bar := renderBar(memPct, 20)
	label := v.clusterMetrics.MemUsage
	if v.clusterMetrics.MemCapacity != "" {
		label += "  (" + v.clusterMetrics.MemCapacity + ")"
	}
	return baseStyle.Render("  MEM ") + bar + baseStyle.Render(" "+label)
}

func (v *HealthView) renderPodCountLine() string {
	var running, pending, failed, succeeded int
	for _, p := range v.pods {
		switch p.Phase {
		case "Running":
			running++
		case "Pending":
			pending++
		case "Failed":
			failed++
		case "Succeeded":
			succeeded++
		}
	}

	parts := theme.Styles.StatusHealthy.Render(fmt.Sprintf("Pods  %s %d Running", theme.IconRunning, running))
	if pending > 0 {
		parts += "  " + theme.Styles.StatusPending.Render(fmt.Sprintf("%s %d Pending", theme.IconPending, pending))
	}
	if failed > 0 {
		parts += "  " + theme.Styles.StatusError.Render(fmt.Sprintf("%s %d Failed", theme.IconError, failed))
	}
	if succeeded > 0 {
		parts += "  " + theme.Styles.StatusUnknown.Render(fmt.Sprintf("%s %d Succ", theme.IconSuccess, succeeded))
	}
	return parts
}

func (v *HealthView) renderWorkloadCountLine() string {
	baseStyle := lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorBackground)

	var deployReady, deployTotal int
	for _, d := range v.deployments {
		deployTotal++
		if d.ReadyReplicas >= d.Replicas && d.Replicas > 0 {
			deployReady++
		}
	}
	var stsReady, stsTotal int
	for _, s := range v.statefulsets {
		stsTotal++
		if s.ReadyReplicas >= s.Replicas && s.Replicas > 0 {
			stsReady++
		}
	}
	var dsReady, dsTotal int
	for _, d := range v.daemonsets {
		dsTotal++
		if d.ReadyNumber >= d.DesiredNumber {
			dsReady++
		}
	}
	var jobsOK, jobsFail int
	for _, j := range v.jobs {
		switch j.Status {
		case "Complete", "Completed":
			jobsOK++
		case "Failed":
			jobsFail++
		default:
			if j.Succeeded > 0 && j.Failed == 0 {
				jobsOK++
			} else if j.Failed > 0 {
				jobsFail++
			}
		}
	}

	statusColor := func(ready, total int) lipgloss.Style {
		if ready < total {
			return theme.Styles.StatusWarning
		}
		return theme.Styles.StatusHealthy
	}

	parts := statusColor(deployReady, deployTotal).Render(fmt.Sprintf("Deploy %s %d/%d", theme.IconRunning, deployReady, deployTotal))
	if stsTotal > 0 {
		parts += baseStyle.Render("  ") + statusColor(stsReady, stsTotal).Render(fmt.Sprintf("STS %s %d/%d", theme.IconRunning, stsReady, stsTotal))
	}
	if dsTotal > 0 {
		parts += baseStyle.Render("  ") + statusColor(dsReady, dsTotal).Render(fmt.Sprintf("DS %s %d/%d", theme.IconRunning, dsReady, dsTotal))
	}
	if len(v.jobs) > 0 {
		jobStr := fmt.Sprintf("Jobs %d/%d", jobsOK, jobsOK+jobsFail)
		parts += baseStyle.Render("  ") + baseStyle.Render(jobStr)
	}

	return parts
}

// --- Nodes section ---

func (v *HealthView) renderNodes(width int) string {
	var b strings.Builder

	var readyCount, notReadyCount int
	for _, n := range v.nodes {
		if n.Status == "Ready" {
			readyCount++
		} else {
			notReadyCount++
		}
	}
	totalNodes := len(v.nodes)

	headerText := fmt.Sprintf("Nodes (%d total: %d Ready", totalNodes, readyCount)
	if notReadyCount > 0 {
		headerText += fmt.Sprintf(", %d NotReady", notReadyCount)
	}
	headerText += ")"
	b.WriteString(renderSectionHeader(headerText, width, v.sectionFocus == sectionNodes, -1))
	b.WriteString("\n")

	if totalNodes == 0 {
		line := theme.Styles.StatusUnknown.Render("  No nodes found")
		b.WriteString(theme.PadToWidth(line, width, theme.ColorBackground))
		b.WriteString("\n")
		return b.String()
	}

	cols := nodeColWidths(width)

	// Column header — 2-char prefix matches indicator width, ColorAccent matches table headers
	hdrStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Background(theme.ColorBackground).
		Bold(true)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	gap := bgStyle.Render(" ")
	hdr := bgStyle.Render("  ") +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.name, "NAME")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.status, "STATUS")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.roles, "ROLES")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.cpu, "CPU")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.mem, "MEM")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.pods, "PODS")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.ip, "IP")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.version, "VERSION"))
	b.WriteString(theme.PadToWidth(hdr, width, theme.ColorBackground))
	b.WriteString("\n")

	showCursor := v.itemMode && v.sectionFocus == sectionNodes

	for i, entry := range v.displayedNodes {
		isSel := showCursor && i == v.nodeCursor
		b.WriteString(v.renderNodeRow(isSel, entry, cols, width))
		b.WriteString("\n")
	}

	if remaining := totalNodes - len(v.displayedNodes); remaining > 0 {
		moreStyle := lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Background(theme.ColorBackground)
		moreLine := moreStyle.Render(fmt.Sprintf("  ... and %d more nodes", remaining))
		b.WriteString(theme.PadToWidth(moreLine, width, theme.ColorBackground))
		b.WriteString("\n")
	}

	return b.String()
}

func (v *HealthView) renderNodeRow(isSelected bool, e healthNodeEntry, cols nodeCols, width int) string {
	n := e.node
	nameStyle := healthCellStyle(theme.ColorText, isSelected)
	statusStyle := healthCellStyle(theme.ColorSuccess, isSelected)
	if n.Status != "Ready" {
		statusStyle = healthCellStyle(theme.ColorError, isSelected)
	}
	gap := healthGap(isSelected)

	rolesStr := strings.Join(n.Roles, ",")
	if rolesStr == "" {
		rolesStr = "<none>"
	}

	cpuStr := "-"
	if e.cpuPct >= 0 {
		cpuStr = fmt.Sprintf("%d%%", e.cpuPct)
	}
	memStr := "-"
	if e.memPct >= 0 {
		memStr = fmt.Sprintf("%d%%", e.memPct)
	}

	ip := n.InternalIP
	if ip == "" {
		ip = "-"
	}
	ver := n.Version
	if ver == "" {
		ver = "-"
	}

	line := healthIndicator(isSelected) +
		nameStyle.Render(fmt.Sprintf("%-*s", cols.name, theme.TruncateString(n.Name, cols.name))) + gap +
		statusStyle.Render(fmt.Sprintf("%-*s", cols.status, n.Status)) + gap +
		nameStyle.Render(fmt.Sprintf("%-*s", cols.roles, theme.TruncateString(rolesStr, cols.roles))) + gap +
		nameStyle.Render(fmt.Sprintf("%-*s", cols.cpu, cpuStr)) + gap +
		nameStyle.Render(fmt.Sprintf("%-*s", cols.mem, memStr)) + gap +
		nameStyle.Render(fmt.Sprintf("%-*s", cols.pods, fmt.Sprintf("%d", n.PodCount))) + gap +
		nameStyle.Render(fmt.Sprintf("%-*s", cols.ip, theme.TruncateString(ip, cols.ip))) + gap +
		nameStyle.Render(fmt.Sprintf("%-*s", cols.version, theme.TruncateString(ver, cols.version)))

	return healthPadRow(line, width, isSelected)
}

// --- Unhealthy Workloads section ---

func (v *HealthView) renderUnhealthyWorkloads(width int) string {
	var b strings.Builder

	title := fmt.Sprintf("Unhealthy Workloads (%d)", len(v.unhealthyWorkloads))
	b.WriteString(renderSectionHeader(title, width, v.sectionFocus == sectionUnhealthy, len(v.unhealthyWorkloads)))
	b.WriteString("\n")

	if len(v.unhealthyWorkloads) == 0 {
		return b.String()
	}

	showCursor := v.itemMode && v.sectionFocus == sectionUnhealthy

	cols := unhealthyColWidths(width)

	// Column header
	hdrStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Background(theme.ColorBackground).
		Bold(true)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	gap := bgStyle.Render(" ")
	hdr := bgStyle.Render("  ") +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.kind, "KIND")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.name, "NAME")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.ready, "READY")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.age, "AGE"))
	b.WriteString(theme.PadToWidth(hdr, width, theme.ColorBackground))
	b.WriteString("\n")

	for i, w := range v.unhealthyWorkloads {
		isSel := showCursor && i == v.unhealthyCursor
		nameStyle := healthCellStyle(theme.ColorText, isSel)
		mutedStyle := healthCellStyle(theme.ColorMuted, isSel)
		gap := healthGap(isSel)

		workloadName := v.qualifiedName(w.Namespace, w.Name)

		readyFg := theme.ColorWarning
		if w.Ready == 0 {
			readyFg = theme.ColorError
		}
		readyStyle := healthCellStyle(readyFg, isSel)

		line := healthIndicator(isSel) +
			nameStyle.Render(fmt.Sprintf("%-*s", cols.kind, w.Kind)) + gap +
			nameStyle.Render(fmt.Sprintf("%-*s", cols.name, theme.TruncateString(workloadName, cols.name))) + gap +
			readyStyle.Render(fmt.Sprintf("%-*s", cols.ready, fmt.Sprintf("%d/%d", w.Ready, w.Desired))) + gap +
			mutedStyle.Render(fmt.Sprintf("%-*s", cols.age, theme.FormatAge(w.Age)))

		b.WriteString(healthPadRow(line, width, isSel))
		b.WriteString("\n")
	}

	return b.String()
}

// --- Failed Jobs section ---

func (v *HealthView) renderFailedJobs(width int) string {
	var b strings.Builder

	title := fmt.Sprintf("Failed Jobs (%d)", len(v.failedJobs))
	b.WriteString(renderSectionHeader(title, width, v.sectionFocus == sectionFailedJobs, len(v.failedJobs)))
	b.WriteString("\n")

	if len(v.failedJobs) == 0 {
		return b.String()
	}

	showCursor := v.itemMode && v.sectionFocus == sectionFailedJobs

	cols := failedJobColWidths(width)

	// Column header
	hdrStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Background(theme.ColorBackground).
		Bold(true)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	gap := bgStyle.Render(" ")
	hdr := bgStyle.Render("  ") +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.name, "JOB")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.status, "STATUS")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.completions, "COMPLETIONS")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.age, "AGE"))
	b.WriteString(theme.PadToWidth(hdr, width, theme.ColorBackground))
	b.WriteString("\n")

	for i, j := range v.failedJobs {
		isSel := showCursor && i == v.failedJobCursor
		nameStyle := healthCellStyle(theme.ColorText, isSel)
		mutedStyle := healthCellStyle(theme.ColorMuted, isSel)
		gap := healthGap(isSel)

		jobName := v.qualifiedName(j.Namespace, j.Name)

		status := j.Status
		if status == "" {
			if j.Failed > 0 {
				status = "Failed"
			} else {
				status = "Running"
			}
		}

		statusFg := theme.ColorError
		if j.Active > 0 {
			statusFg = theme.ColorWarning
		}
		statusStyle := healthCellStyle(statusFg, isSel)

		completions := fmt.Sprintf("%d/%d (%d fail)", j.Succeeded, j.Completions, j.Failed)

		line := healthIndicator(isSel) +
			nameStyle.Render(fmt.Sprintf("%-*s", cols.name, theme.TruncateString(jobName, cols.name))) + gap +
			statusStyle.Render(fmt.Sprintf("%-*s", cols.status, theme.TruncateString(status, cols.status))) + gap +
			nameStyle.Render(fmt.Sprintf("%-*s", cols.completions, theme.TruncateString(completions, cols.completions))) + gap +
			mutedStyle.Render(fmt.Sprintf("%-*s", cols.age, theme.FormatAge(j.Age)))

		b.WriteString(healthPadRow(line, width, isSel))
		b.WriteString("\n")
	}

	return b.String()
}

// --- Problem Pods section ---

func (v *HealthView) renderProblemPods(width int) string {
	var b strings.Builder

	title := fmt.Sprintf("Problem Pods (%d)", len(v.problemPods))
	b.WriteString(renderSectionHeader(title, width, v.sectionFocus == sectionProblems, len(v.problemPods)))
	b.WriteString("\n")

	if len(v.problemPods) == 0 {
		return b.String()
	}

	showCursor := v.itemMode && v.sectionFocus == sectionProblems

	cols := problemColWidths(width)

	// Column header
	hdrStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Background(theme.ColorBackground).
		Bold(true)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	gap := bgStyle.Render(" ")
	hdr := bgStyle.Render("  ") +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.name, "POD")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.ready, "READY")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.status, "STATUS")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.age, "AGE"))
	b.WriteString(theme.PadToWidth(hdr, width, theme.ColorBackground))
	b.WriteString("\n")

	for i, p := range v.problemPods {
		isSel := showCursor && i == v.problemCursor
		nameStyle := healthCellStyle(theme.ColorText, isSel)
		mutedStyle := healthCellStyle(theme.ColorMuted, isSel)
		gap := healthGap(isSel)

		podName := v.qualifiedName(p.Namespace, p.Name)

		// Show the most informative status: container reason if available, otherwise phase
		status := p.Phase
		for _, c := range p.Containers {
			if c.StateReason != "" && c.State != "running" {
				status = c.StateReason
				break
			}
		}

		statusFg := theme.ColorWarning
		if p.Phase == "Failed" || status == "CrashLoopBackOff" || status == "ImagePullBackOff" || status == "ErrImagePull" {
			statusFg = theme.ColorError
		}
		statusStyle := healthCellStyle(statusFg, isSel)

		// Ready column — use pre-computed Ready string (e.g. "1/2")
		ready := p.Ready
		if ready == "" {
			ready = "-"
		}
		readyFg := theme.ColorMuted
		if ready != "-" && !strings.HasPrefix(ready, "0/") {
			readyFg = theme.ColorText
		}
		readyStyle := healthCellStyle(readyFg, isSel)

		line := healthIndicator(isSel) +
			nameStyle.Render(fmt.Sprintf("%-*s", cols.name, theme.TruncateString(podName, cols.name))) + gap +
			readyStyle.Render(fmt.Sprintf("%-*s", cols.ready, ready)) + gap +
			statusStyle.Render(theme.StatusIconPrefix(status)+" "+fmt.Sprintf("%-*s", cols.status, theme.TruncateString(status, cols.status))) + gap +
			mutedStyle.Render(fmt.Sprintf("%-*s", cols.age, theme.FormatAge(p.Age)))

		b.WriteString(healthPadRow(line, width, isSel))
		b.WriteString("\n")
	}

	return b.String()
}

// --- Pending PVCs section ---

func (v *HealthView) renderPendingPVCs(width int) string {
	var b strings.Builder

	title := fmt.Sprintf("Pending PVCs (%d)", len(v.pendingPVCs))
	b.WriteString(renderSectionHeader(title, width, v.sectionFocus == sectionPendingPVCs, len(v.pendingPVCs)))
	b.WriteString("\n")

	if len(v.pendingPVCs) == 0 {
		return b.String()
	}

	showCursor := v.itemMode && v.sectionFocus == sectionPendingPVCs

	cols := pendingPVCColWidths(width)

	// Column header
	hdrStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Background(theme.ColorBackground).
		Bold(true)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	gap := bgStyle.Render(" ")
	hdr := bgStyle.Render("  ") +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.name, "PVC")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.storageClass, "STORAGE CLASS")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.age, "AGE"))
	b.WriteString(theme.PadToWidth(hdr, width, theme.ColorBackground))
	b.WriteString("\n")

	for i, pvc := range v.pendingPVCs {
		isSel := showCursor && i == v.pendingPVCCursor
		nameStyle := healthCellStyle(theme.ColorText, isSel)
		mutedStyle := healthCellStyle(theme.ColorMuted, isSel)
		gap := healthGap(isSel)

		pvcName := v.qualifiedName(pvc.Namespace, pvc.Name)

		sc := pvc.StorageClass
		if sc == "" {
			sc = "<default>"
		}

		ageFg := theme.ColorWarning
		ageStyle := healthCellStyle(ageFg, isSel)

		line := healthIndicator(isSel) +
			nameStyle.Render(fmt.Sprintf("%-*s", cols.name, theme.TruncateString(pvcName, cols.name))) + gap +
			mutedStyle.Render(fmt.Sprintf("%-*s", cols.storageClass, theme.TruncateString(sc, cols.storageClass))) + gap +
			ageStyle.Render(fmt.Sprintf("%-*s", cols.age, theme.FormatAge(pvc.Age)))

		b.WriteString(healthPadRow(line, width, isSel))
		b.WriteString("\n")
	}

	return b.String()
}

// --- Restarts section ---

func (v *HealthView) renderRestarts(width int) string {
	var b strings.Builder

	title := fmt.Sprintf("Top Restarting Pods (%d)", len(v.restartingPods))
	b.WriteString(renderSectionHeader(title, width, v.sectionFocus == sectionRestarts, len(v.restartingPods)))
	b.WriteString("\n")

	if len(v.restartingPods) == 0 {
		return b.String()
	}

	showCursor := v.itemMode && v.sectionFocus == sectionRestarts

	cols := restartColWidths(width)

	// Column header
	hdrStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Background(theme.ColorBackground).
		Bold(true)
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	gap := bgStyle.Render(" ")
	hdr := bgStyle.Render("  ") +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.name, "POD")) + gap +
		hdrStyle.Render(fmt.Sprintf("%*s", cols.restarts, "RESTARTS")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.reason, "REASON")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.status, "STATUS")) + gap +
		hdrStyle.Render(fmt.Sprintf("%-*s", cols.lastRestart, "LAST RESTART"))
	b.WriteString(theme.PadToWidth(hdr, width, theme.ColorBackground))
	b.WriteString("\n")

	for i, p := range v.restartingPods {
		isSel := showCursor && i == v.restartCursor
		nameStyle := healthCellStyle(theme.ColorText, isSel)
		mutedStyle := healthCellStyle(theme.ColorMuted, isSel)
		gap := healthGap(isSel)

		countFg := theme.ColorWarning
		if p.Restarts >= 100 {
			countFg = theme.ColorError
		}
		countStyle := healthCellStyle(countFg, isSel)

		podName := v.qualifiedName(p.Namespace, p.Name)

		// Find the container with the most restarts and its termination reason
		reason := ""
		var topCount int32
		var topLastTerminated time.Time
		for _, c := range p.Containers {
			if c.RestartCount > topCount {
				topCount = c.RestartCount
				reason = c.StateReason
				topLastTerminated = c.LastTerminatedAt
			}
		}

		reasonFg := theme.ColorMuted
		if reason == "OOMKilled" {
			reasonFg = theme.ColorError
		} else if reason == "CrashLoopBackOff" || reason == "Error" {
			reasonFg = theme.ColorWarning
		}
		reasonStyle := healthCellStyle(reasonFg, isSel)
		if reason == "" {
			reason = "-"
		}

		lastRestart := "-"
		if !topLastTerminated.IsZero() {
			lastRestart = theme.FormatAge(time.Since(topLastTerminated))
		}

		line := healthIndicator(isSel) +
			nameStyle.Render(fmt.Sprintf("%-*s", cols.name, theme.TruncateString(podName, cols.name))) + gap +
			countStyle.Render(fmt.Sprintf("%*d", cols.restarts, p.Restarts)) + gap +
			reasonStyle.Render(fmt.Sprintf("%-*s", cols.reason, theme.TruncateString(reason, cols.reason))) + gap +
			mutedStyle.Render(theme.StatusIconPrefix(p.Phase)+" "+fmt.Sprintf("%-*s", cols.status, theme.TruncateString(p.Phase, cols.status))) + gap +
			mutedStyle.Render(fmt.Sprintf("%-*s", cols.lastRestart, lastRestart))

		b.WriteString(healthPadRow(line, width, isSel))
		b.WriteString("\n")
	}

	return b.String()
}

// --- Issues section ---

func (v *HealthView) renderIssues(width int) string {
	var b strings.Builder

	title := fmt.Sprintf("Top Issues (%d)", len(v.diagnoses))
	b.WriteString(renderSectionHeader(title, width, v.sectionFocus == sectionIssues, len(v.diagnoses)))
	b.WriteString("\n")

	if len(v.diagnoses) == 0 {
		return b.String()
	}

	showCursor := v.itemMode && v.sectionFocus == sectionIssues

	cols := issueColWidths(width)

	for i, d := range v.diagnoses {
		isSel := showCursor && i == v.issueCursor
		gap := healthGap(isSel)

		severityFg := theme.ColorMuted
		severityLabel := "INFO"
		icon := theme.IconPending
		switch d.Severity {
		case analyzer.SeverityCritical:
			severityFg = theme.ColorError
			severityLabel = "CRIT"
			icon = theme.IconError
		case analyzer.SeverityWarning:
			severityFg = theme.ColorWarning
			severityLabel = "WARN"
			icon = theme.IconWarning
		case analyzer.SeverityInfo:
			severityFg = theme.ColorHighlight
			severityLabel = "INFO"
			icon = theme.IconPending
		}
		severityStyle := healthCellStyle(severityFg, isSel)

		resourceStyle := healthCellStyle(theme.ColorText, isSel)
		problemStyle := healthCellStyle(theme.ColorMuted, isSel)

		resourceName := v.qualifiedName(d.Namespace, d.ResourceName)

		line := healthIndicator(isSel) +
			severityStyle.Render(icon+" "+fmt.Sprintf("%-*s", cols.severity, severityLabel)) + gap +
			resourceStyle.Render(fmt.Sprintf("%-*s", cols.resource, theme.TruncateString(resourceName, cols.resource))) + gap +
			problemStyle.Render(fmt.Sprintf("%-*s", cols.problem, theme.TruncateString(d.Problem, cols.problem)))

		b.WriteString(healthPadRow(line, width, isSel))
		b.WriteString("\n")
	}

	return b.String()
}

// --- Events section ---

func (v *HealthView) renderEvents(width int) string {
	var b strings.Builder

	title := "Recent Events"
	b.WriteString(renderSectionHeader(title, width, v.sectionFocus == sectionEvents, -1))
	b.WriteString("\n")

	if len(v.events) == 0 {
		line := theme.Styles.StatusUnknown.Render("  No recent events")
		b.WriteString(theme.PadToWidth(line, width, theme.ColorBackground))
		b.WriteString("\n")
		return b.String()
	}

	maxEvents := min(len(v.events), 15)
	showCursor := v.itemMode && v.sectionFocus == sectionEvents

	cols := eventColWidths(width)

	for i := 0; i < maxEvents; i++ {
		e := v.events[i]
		isSel := showCursor && i == v.eventCursor
		gap := healthGap(isSel)

		ago := theme.FormatAge(time.Since(e.LastSeen))
		agoStyle := healthCellStyle(theme.ColorMuted, isSel)

		typeFg := theme.ColorSuccess
		typeIcon := theme.IconRunning
		typeLabel := "Normal"
		if e.Type == "Warning" {
			typeFg = theme.ColorWarning
			typeIcon = theme.IconWarning
			typeLabel = "Warning"
		}
		typeStyle := healthCellStyle(typeFg, isSel)

		resourceStyle := healthCellStyle(theme.ColorText, isSel)
		resourceName := v.qualifiedName(e.Namespace, e.ObjectName)

		msgStyle := healthCellStyle(theme.ColorMuted, isSel)

		fullMsg := theme.TruncateString(e.Reason+": "+e.Message, cols.message)

		line := healthIndicator(isSel) +
			agoStyle.Render(fmt.Sprintf("%-*s", cols.ago, ago+" ago")) + gap +
			typeStyle.Render(typeIcon+" "+fmt.Sprintf("%-*s", cols.typeLabel, typeLabel)) + gap +
			resourceStyle.Render(fmt.Sprintf("%-*s", cols.resource, theme.TruncateString(resourceName, cols.resource))) + gap +
			msgStyle.Render(fmt.Sprintf("%-*s", cols.message, fullMsg))

		b.WriteString(healthPadRow(line, width, isSel))
		b.WriteString("\n")
	}

	return b.String()
}
