package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bijaya/kview/internal/ui/theme"
)

// TabCategory represents a group of related resources
type TabCategory struct {
	Name      string           // Category name (e.g., "Workloads")
	Resources []string         // Resource names in this category
	ViewTypes []theme.ViewType // Actual ViewType for each resource
	StartIdx  int              // Kept for backward compat (first resource's ViewType)
}

// CategoryTabs provides two-tier navigation with categories and resources
type CategoryTabs struct {
	width            int
	categories       []TabCategory
	activeCategoryIdx int
	activeResourceIdx int // Index within the current category
	globalActiveIdx   int // Global index across all resources
	highlightActive   bool // true when activeView is a category resource
}

// Categories defines the resource groupings
var Categories = []TabCategory{
	{Name: "Workloads", Resources: []string{"Pods", "Deploy", "RS", "DS", "STS", "Jobs", "CJ", "HPA"},
		ViewTypes: []theme.ViewType{theme.ViewPods, theme.ViewDeployments, theme.ViewReplicaSets, theme.ViewDaemonSets, theme.ViewStatefulSets, theme.ViewJobs, theme.ViewCronJobs, theme.ViewHPAs},
		StartIdx: 0},
	{Name: "Network", Resources: []string{"Svc", "Ing"},
		ViewTypes: []theme.ViewType{theme.ViewServices, theme.ViewIngresses},
		StartIdx: 2},
	{Name: "Config", Resources: []string{"CM", "Sec", "PVC"},
		ViewTypes: []theme.ViewType{theme.ViewConfigMaps, theme.ViewSecrets, theme.ViewPVCs},
		StartIdx: 3},
	{Name: "Cluster", Resources: []string{"Nodes", "Events", "PV", "RB"},
		ViewTypes: []theme.ViewType{theme.ViewNodes, theme.ViewEvents, theme.ViewPVs, theme.ViewRoleBindings},
		StartIdx: 8},
	{Name: "Helm", Resources: []string{"Rel"},
		ViewTypes: []theme.ViewType{theme.ViewHelmReleases},
		StartIdx: 0},
}

// NewCategoryTabs creates a new category tabs component
func NewCategoryTabs() *CategoryTabs {
	return &CategoryTabs{
		width:            80,
		categories:       Categories,
		activeCategoryIdx: 0,
		activeResourceIdx: 0,
		globalActiveIdx:   0,
	}
}

// SetWidth sets the component width
func (c *CategoryTabs) SetWidth(width int) {
	c.width = width
}

// SetActiveByGlobalIndex sets the active resource by global index (ViewType value)
func (c *CategoryTabs) SetActiveByGlobalIndex(globalIdx int) {
	c.globalActiveIdx = globalIdx

	// Search ViewTypes arrays to find matching category/resource
	for catIdx, cat := range c.categories {
		for resIdx, vt := range cat.ViewTypes {
			if int(vt) == globalIdx {
				c.activeCategoryIdx = catIdx
				c.activeResourceIdx = resIdx
				c.highlightActive = true
				return
			}
		}
	}
	c.highlightActive = false
}

// IsHighlightActive returns whether the current view is a category resource
func (c *CategoryTabs) IsHighlightActive() bool {
	return c.highlightActive
}

// GlobalActiveIndex returns the current global active index
func (c *CategoryTabs) GlobalActiveIndex() int {
	return c.globalActiveIdx
}

// ActiveCategory returns the current category index
func (c *CategoryTabs) ActiveCategory() int {
	return c.activeCategoryIdx
}

// ActiveResourceIndex returns the active resource index within the current category
func (c *CategoryTabs) ActiveResourceIndex() int {
	return c.activeResourceIdx
}

// NextCategory moves to the next category
func (c *CategoryTabs) NextCategory() int {
	if c.activeCategoryIdx < len(c.categories)-1 {
		c.activeCategoryIdx++
		c.activeResourceIdx = 0
		if len(c.categories[c.activeCategoryIdx].ViewTypes) > 0 {
			c.globalActiveIdx = int(c.categories[c.activeCategoryIdx].ViewTypes[0])
		}
	}
	return c.globalActiveIdx
}

// PrevCategory moves to the previous category
func (c *CategoryTabs) PrevCategory() int {
	if c.activeCategoryIdx > 0 {
		c.activeCategoryIdx--
		c.activeResourceIdx = 0
		if len(c.categories[c.activeCategoryIdx].ViewTypes) > 0 {
			c.globalActiveIdx = int(c.categories[c.activeCategoryIdx].ViewTypes[0])
		}
	}
	return c.globalActiveIdx
}

// NextResource moves to the next resource within the category (wraps to next category)
func (c *CategoryTabs) NextResource() int {
	cat := c.categories[c.activeCategoryIdx]
	if c.activeResourceIdx < len(cat.Resources)-1 {
		c.activeResourceIdx++
	} else if c.activeCategoryIdx < len(c.categories)-1 {
		// Move to next category
		c.activeCategoryIdx++
		c.activeResourceIdx = 0
	}
	c.globalActiveIdx = int(c.categories[c.activeCategoryIdx].ViewTypes[c.activeResourceIdx])
	return c.globalActiveIdx
}

// PrevResource moves to the previous resource within the category (wraps to prev category)
func (c *CategoryTabs) PrevResource() int {
	if c.activeResourceIdx > 0 {
		c.activeResourceIdx--
	} else if c.activeCategoryIdx > 0 {
		// Move to previous category
		c.activeCategoryIdx--
		cat := c.categories[c.activeCategoryIdx]
		c.activeResourceIdx = len(cat.Resources) - 1
	}
	c.globalActiveIdx = int(c.categories[c.activeCategoryIdx].ViewTypes[c.activeResourceIdx])
	return c.globalActiveIdx
}

// SelectResourceByNumber selects a resource by its number key (1-9) within current category
func (c *CategoryTabs) SelectResourceByNumber(num int) (int, bool) {
	cat := c.categories[c.activeCategoryIdx]
	if num >= 1 && num <= len(cat.Resources) {
		c.activeResourceIdx = num - 1
		c.globalActiveIdx = int(cat.ViewTypes[c.activeResourceIdx])
		return c.globalActiveIdx, true
	}
	return c.globalActiveIdx, false
}

// Update handles key events for category navigation
func (c *CategoryTabs) Update(msg tea.Msg) (*CategoryTabs, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, theme.DefaultKeyMap().Left):
			return c, func() tea.Msg {
				return theme.SwitchViewMsg{View: theme.ViewType(c.PrevCategory())}
			}
		case key.Matches(msg, theme.DefaultKeyMap().Right):
			return c, func() tea.Msg {
				return theme.SwitchViewMsg{View: theme.ViewType(c.NextCategory())}
			}
		}
	}
	return c, nil
}

// View renders both category row and resource row
func (c *CategoryTabs) View() string {
	categoryRow := c.renderCategoryRow()
	resourceRow := c.renderResourceRow()

	return categoryRow + "\n" + resourceRow
}

// ViewCategoryRow renders just the category row
func (c *CategoryTabs) ViewCategoryRow() string {
	return c.renderCategoryRow()
}

// ViewResourceRow renders just the resource row
func (c *CategoryTabs) ViewResourceRow() string {
	return c.renderResourceRow()
}

func (c *CategoryTabs) renderCategoryRow() string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	var parts []string

	for i, cat := range c.categories {
		var rendered string
		if i == c.activeCategoryIdx {
			indicator := theme.Styles.CategoryIndicator.Render("► ")
			name := theme.Styles.CategoryItemActive.Render(cat.Name)
			rendered = indicator + name
		} else {
			// Space to align with indicator
			rendered = bgStyle.Render("  ") + theme.Styles.CategoryItem.Render(cat.Name)
		}
		parts = append(parts, rendered)
	}

	content := strings.Join(parts, bgStyle.Render("  "))

	// Pad to full width
	contentWidth := lipgloss.Width(content)
	if contentWidth < c.width {
		content = content + bgStyle.Render(strings.Repeat(" ", c.width-contentWidth))
	}

	return theme.Styles.CategoryRow.Width(c.width).Render(content)
}

func (c *CategoryTabs) renderResourceRow() string {
	bgStyle := lipgloss.NewStyle().Background(theme.ColorBackground)
	cat := c.categories[c.activeCategoryIdx]
	var parts []string

	for i, res := range cat.Resources {
		number := fmt.Sprintf("[%d]", i+1)
		numberStyle := theme.Styles.TabBarNumber

		var itemStyle lipgloss.Style
		if i == c.activeResourceIdx {
			itemStyle = theme.Styles.ResourceItemActive
		} else {
			itemStyle = theme.Styles.ResourceItem
		}

		tab := numberStyle.Render(number) + itemStyle.Render(res)
		parts = append(parts, tab)
	}

	content := strings.Join(parts, bgStyle.Render(" "))

	// Pad to full width
	contentWidth := lipgloss.Width(content)
	if contentWidth < c.width {
		content = content + bgStyle.Render(strings.Repeat(" ", c.width-contentWidth))
	}

	return theme.Styles.ResourceRow.Width(c.width).Render(content)
}
