package rules

import (
	"strings"
	"testing"

	"github.com/bijaya/kview/internal/analyzer"
	"github.com/bijaya/kview/internal/k8s"
)

// makePod builds a minimal PodInfo fixture for rule tests
func makePod(name, uid string, containers ...k8s.ContainerInfo) k8s.PodInfo {
	return k8s.PodInfo{
		Resource: k8s.Resource{
			UID:       uid,
			Kind:      "Pod",
			Namespace: "default",
			Name:      name,
		},
		Phase:      "Running",
		Containers: containers,
	}
}

func healthyContainer(name string) k8s.ContainerInfo {
	return k8s.ContainerInfo{
		Name:  name,
		Image: "nginx:1.25",
		Ready: true,
		State: "Running",
	}
}

func TestCrashLoopRule(t *testing.T) {
	rule := &CrashLoopRule{}

	if rule.Name() != "crashloop-backoff" {
		t.Errorf("Name() = %q, want %q", rule.Name(), "crashloop-backoff")
	}

	tests := []struct {
		name          string
		pods          []k8s.PodInfo
		wantCount     int
		wantID        string
		wantProblem   string
		wantRootCause []string
	}{
		{
			name: "container in CrashLoopBackOff produces diagnosis",
			pods: []k8s.PodInfo{
				makePod("web-abc", "uid-1", k8s.ContainerInfo{
					Name:         "app",
					Image:        "myapp:v1",
					State:        "Waiting",
					StateReason:  "CrashLoopBackOff",
					StateMessage: "back-off 5m0s restarting failed container",
					RestartCount: 7,
				}),
			},
			wantCount:   1,
			wantID:      "crashloop-uid-1-app",
			wantProblem: "Container app is in CrashLoopBackOff",
			wantRootCause: []string{
				"crashed 7 times",
				"back-off 5m0s restarting failed container",
			},
		},
		{
			name: "healthy running pod produces no diagnosis",
			pods: []k8s.PodInfo{
				makePod("web-ok", "uid-2", healthyContainer("app")),
			},
			wantCount: 0,
		},
		{
			name: "only crashing container flagged in multi-container pod",
			pods: []k8s.PodInfo{
				makePod("multi", "uid-3",
					healthyContainer("sidecar"),
					k8s.ContainerInfo{
						Name:        "app",
						State:       "Waiting",
						StateReason: "CrashLoopBackOff",
					},
				),
			},
			wantCount:   1,
			wantID:      "crashloop-uid-3-app",
			wantProblem: "Container app is in CrashLoopBackOff",
		},
		{
			name:      "no pods produces no diagnoses",
			pods:      nil,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnoses := rule.Analyze(nil, tt.pods, nil)
			if len(diagnoses) != tt.wantCount {
				t.Fatalf("Analyze() returned %d diagnoses, want %d", len(diagnoses), tt.wantCount)
			}
			if tt.wantCount == 0 {
				return
			}

			d := diagnoses[0]
			if d.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", d.ID, tt.wantID)
			}
			if d.Severity != analyzer.SeverityCritical {
				t.Errorf("Severity = %q, want %q", d.Severity, analyzer.SeverityCritical)
			}
			if d.ResourceKind != "Pod" {
				t.Errorf("ResourceKind = %q, want %q", d.ResourceKind, "Pod")
			}
			if d.ResourceName != tt.pods[0].Name {
				t.Errorf("ResourceName = %q, want %q", d.ResourceName, tt.pods[0].Name)
			}
			if d.Namespace != "default" {
				t.Errorf("Namespace = %q, want %q", d.Namespace, "default")
			}
			if d.Problem != tt.wantProblem {
				t.Errorf("Problem = %q, want %q", d.Problem, tt.wantProblem)
			}
			for _, want := range tt.wantRootCause {
				if !strings.Contains(d.RootCause, want) {
					t.Errorf("RootCause missing %q\ngot: %s", want, d.RootCause)
				}
			}
			if len(d.Suggestions) == 0 {
				t.Error("expected at least one suggestion")
			}
		})
	}
}

func TestOOMKilledRule(t *testing.T) {
	rule := &OOMKilledRule{}

	if rule.Name() != "oom-killed" {
		t.Errorf("Name() = %q, want %q", rule.Name(), "oom-killed")
	}

	tests := []struct {
		name          string
		pods          []k8s.PodInfo
		wantCount     int
		wantID        string
		wantProblem   string
		wantRootCause []string
	}{
		{
			name: "OOMKilled container produces diagnosis",
			pods: []k8s.PodInfo{
				makePod("worker-xyz", "uid-10", k8s.ContainerInfo{
					Name:         "worker",
					Image:        "worker:v2",
					State:        "Terminated",
					StateReason:  "OOMKilled",
					RestartCount: 3,
				}),
			},
			wantCount:   1,
			wantID:      "oom-uid-10-worker",
			wantProblem: "Container worker was OOM killed",
			wantRootCause: []string{
				"exceeded its memory limit",
				"restarted 3 times",
			},
		},
		{
			name: "OOMKilled with zero restarts omits restart note",
			pods: []k8s.PodInfo{
				makePod("worker-once", "uid-11", k8s.ContainerInfo{
					Name:        "worker",
					State:       "Terminated",
					StateReason: "OOMKilled",
				}),
			},
			wantCount:   1,
			wantID:      "oom-uid-11-worker",
			wantProblem: "Container worker was OOM killed",
		},
		{
			name: "healthy pod not flagged",
			pods: []k8s.PodInfo{
				makePod("worker-ok", "uid-12", healthyContainer("worker")),
			},
			wantCount: 0,
		},
		{
			name: "terminated with Completed reason not flagged",
			pods: []k8s.PodInfo{
				makePod("job-pod", "uid-13", k8s.ContainerInfo{
					Name:        "job",
					State:       "Terminated",
					StateReason: "Completed",
				}),
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnoses := rule.Analyze(nil, tt.pods, nil)
			if len(diagnoses) != tt.wantCount {
				t.Fatalf("Analyze() returned %d diagnoses, want %d", len(diagnoses), tt.wantCount)
			}
			if tt.wantCount == 0 {
				return
			}

			d := diagnoses[0]
			if d.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", d.ID, tt.wantID)
			}
			if d.Severity != analyzer.SeverityCritical {
				t.Errorf("Severity = %q, want %q", d.Severity, analyzer.SeverityCritical)
			}
			if d.ResourceKind != "Pod" {
				t.Errorf("ResourceKind = %q, want %q", d.ResourceKind, "Pod")
			}
			if d.Problem != tt.wantProblem {
				t.Errorf("Problem = %q, want %q", d.Problem, tt.wantProblem)
			}
			for _, want := range tt.wantRootCause {
				if !strings.Contains(d.RootCause, want) {
					t.Errorf("RootCause missing %q\ngot: %s", want, d.RootCause)
				}
			}
			if len(d.Suggestions) == 0 {
				t.Error("expected at least one suggestion")
			}
		})
	}
}

func TestOOMKilledRuleIgnoresRestartNoteWhenNoRestarts(t *testing.T) {
	rule := &OOMKilledRule{}
	pods := []k8s.PodInfo{
		makePod("worker-once", "uid-14", k8s.ContainerInfo{
			Name:        "worker",
			State:       "Terminated",
			StateReason: "OOMKilled",
		}),
	}

	diagnoses := rule.Analyze(nil, pods, nil)
	if len(diagnoses) != 1 {
		t.Fatalf("Analyze() returned %d diagnoses, want 1", len(diagnoses))
	}
	if strings.Contains(diagnoses[0].RootCause, "restarted") {
		t.Errorf("RootCause should not mention restarts when RestartCount is 0\ngot: %s", diagnoses[0].RootCause)
	}
}

func TestImagePullRule(t *testing.T) {
	rule := &ImagePullRule{}

	if rule.Name() != "image-pull-error" {
		t.Errorf("Name() = %q, want %q", rule.Name(), "image-pull-error")
	}

	t.Run("all pull error reasons detected", func(t *testing.T) {
		reasons := []string{
			"ImagePullBackOff",
			"ErrImagePull",
			"ImageInspectError",
			"ErrImageNeverPull",
			"RegistryUnavailable",
		}
		for _, reason := range reasons {
			pods := []k8s.PodInfo{
				makePod("pull-pod", "uid-20", k8s.ContainerInfo{
					Name:        "app",
					Image:       "ghcr.io/example/missing:v9",
					State:       "Waiting",
					StateReason: reason,
				}),
			}
			diagnoses := rule.Analyze(nil, pods, nil)
			if len(diagnoses) != 1 {
				t.Errorf("reason %q: got %d diagnoses, want 1", reason, len(diagnoses))
				continue
			}
			if !strings.Contains(diagnoses[0].RootCause, reason) {
				t.Errorf("reason %q: RootCause missing reason\ngot: %s", reason, diagnoses[0].RootCause)
			}
		}
	})

	tests := []struct {
		name          string
		pods          []k8s.PodInfo
		wantCount     int
		wantID        string
		wantProblem   string
		wantRootCause []string
	}{
		{
			name: "ImagePullBackOff container includes image and message details",
			pods: []k8s.PodInfo{
				makePod("api-pod", "uid-21", k8s.ContainerInfo{
					Name:         "api",
					Image:        "registry.local/api:v3",
					State:        "Waiting",
					StateReason:  "ImagePullBackOff",
					StateMessage: `Back-off pulling image "registry.local/api:v3"`,
				}),
			},
			wantCount:   1,
			wantID:      "imagepull-uid-21-api",
			wantProblem: "Container api cannot pull image",
			wantRootCause: []string{
				"registry.local/api:v3",
				"ImagePullBackOff",
				`Back-off pulling image "registry.local/api:v3"`,
			},
		},
		{
			name: "init container pull error detected with init ID",
			pods: []k8s.PodInfo{
				{
					Resource: k8s.Resource{
						UID:       "uid-22",
						Kind:      "Pod",
						Namespace: "default",
						Name:      "init-pod",
					},
					Containers: []k8s.ContainerInfo{healthyContainer("app")},
					InitContainers: []k8s.ContainerInfo{
						{
							Name:        "setup",
							Image:       "busybox:oops",
							State:       "Waiting",
							StateReason: "ErrImagePull",
						},
					},
				},
			},
			wantCount:     1,
			wantID:        "imagepull-uid-22-init-setup",
			wantProblem:   "Init container setup cannot pull image",
			wantRootCause: []string{"busybox:oops", "ErrImagePull"},
		},
		{
			name: "healthy pod not flagged",
			pods: []k8s.PodInfo{
				makePod("ok-pod", "uid-23", healthyContainer("app")),
			},
			wantCount: 0,
		},
		{
			name: "unrelated waiting reason not flagged",
			pods: []k8s.PodInfo{
				makePod("starting-pod", "uid-24", k8s.ContainerInfo{
					Name:        "app",
					State:       "Waiting",
					StateReason: "ContainerCreating",
				}),
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnoses := rule.Analyze(nil, tt.pods, nil)
			if len(diagnoses) != tt.wantCount {
				t.Fatalf("Analyze() returned %d diagnoses, want %d", len(diagnoses), tt.wantCount)
			}
			if tt.wantCount == 0 {
				return
			}

			d := diagnoses[0]
			if d.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", d.ID, tt.wantID)
			}
			if d.Severity != analyzer.SeverityCritical {
				t.Errorf("Severity = %q, want %q", d.Severity, analyzer.SeverityCritical)
			}
			if d.ResourceKind != "Pod" {
				t.Errorf("ResourceKind = %q, want %q", d.ResourceKind, "Pod")
			}
			if d.Problem != tt.wantProblem {
				t.Errorf("Problem = %q, want %q", d.Problem, tt.wantProblem)
			}
			for _, want := range tt.wantRootCause {
				if !strings.Contains(d.RootCause, want) {
					t.Errorf("RootCause missing %q\ngot: %s", want, d.RootCause)
				}
			}
			if len(d.Suggestions) == 0 {
				t.Error("expected at least one suggestion")
			}
		})
	}
}

func TestRuleSetAnalyzeAggregatesDiagnoses(t *testing.T) {
	rs := NewRuleSet()

	if got := len(rs.Rules()); got != 8 {
		t.Errorf("default rule set has %d rules, want 8", got)
	}

	// One pod crashing, one OOM killed, one with a pull error
	pods := []k8s.PodInfo{
		makePod("crash-pod", "uid-30", k8s.ContainerInfo{
			Name:        "app",
			State:       "Waiting",
			StateReason: "CrashLoopBackOff",
		}),
		makePod("oom-pod", "uid-31", k8s.ContainerInfo{
			Name:        "app",
			State:       "Terminated",
			StateReason: "OOMKilled",
		}),
		makePod("pull-pod", "uid-32", k8s.ContainerInfo{
			Name:        "app",
			Image:       "bad:tag",
			State:       "Waiting",
			StateReason: "ImagePullBackOff",
		}),
	}

	diagnoses := rs.Analyze(nil, pods, nil)

	wantIDs := map[string]bool{
		"crashloop-uid-30-app": false,
		"oom-uid-31-app":       false,
		"imagepull-uid-32-app": false,
	}
	for _, d := range diagnoses {
		if _, ok := wantIDs[d.ID]; ok {
			wantIDs[d.ID] = true
		}
	}
	for id, found := range wantIDs {
		if !found {
			t.Errorf("expected diagnosis %q not found in aggregated results", id)
		}
	}
}
