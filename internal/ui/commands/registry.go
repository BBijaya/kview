package commands

import (
	"sort"
	"strings"
)

// Registry holds all registered commands
type Registry struct {
	commands map[string]Command
	order    []string // Maintain insertion order
}

// NewRegistry creates a new command registry
func NewRegistry() *Registry {
	r := &Registry{
		commands: make(map[string]Command),
	}
	// Register default commands
	for _, cmd := range DefaultCommands() {
		r.Register(cmd)
	}
	return r
}

// Register adds a command to the registry
func (r *Registry) Register(cmd Command) {
	if _, exists := r.commands[cmd.ID]; !exists {
		r.order = append(r.order, cmd.ID)
	}
	r.commands[cmd.ID] = cmd
}

// Unregister removes a command from the registry
func (r *Registry) Unregister(id string) {
	delete(r.commands, id)
	// Remove from order
	for i, orderID := range r.order {
		if orderID == id {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
}

// Get returns a command by ID
func (r *Registry) Get(id string) (Command, bool) {
	cmd, ok := r.commands[id]
	return cmd, ok
}

// All returns all registered commands in order
func (r *Registry) All() []Command {
	commands := make([]Command, 0, len(r.order))
	for _, id := range r.order {
		if cmd, ok := r.commands[id]; ok {
			commands = append(commands, cmd)
		}
	}
	return commands
}

// ByCategory returns commands grouped by category
func (r *Registry) ByCategory() map[string][]Command {
	result := make(map[string][]Command)
	for _, id := range r.order {
		if cmd, ok := r.commands[id]; ok {
			result[cmd.Category] = append(result[cmd.Category], cmd)
		}
	}
	return result
}

// Search searches commands by name or description
func (r *Registry) Search(query string) []Command {
	if query == "" {
		return r.All()
	}

	query = strings.ToLower(query)
	var matches []Command
	var scores []int

	for _, id := range r.order {
		cmd, ok := r.commands[id]
		if !ok {
			continue
		}

		name := strings.ToLower(cmd.Name)
		desc := strings.ToLower(cmd.Description)
		category := strings.ToLower(cmd.Category)

		score := 0
		// Exact match in name
		if strings.HasPrefix(name, query) {
			score = 100
		} else if strings.Contains(name, query) {
			score = 80
		} else if strings.Contains(desc, query) {
			score = 60
		} else if strings.Contains(category, query) {
			score = 40
		}

		if score > 0 {
			matches = append(matches, cmd)
			scores = append(scores, score)
		}
	}

	// Sort by score (descending)
	sort.Slice(matches, func(i, j int) bool {
		return scores[i] > scores[j]
	})

	return matches
}

// FuzzySearch performs fuzzy matching on commands
func (r *Registry) FuzzySearch(query string) []Command {
	if query == "" {
		return r.All()
	}

	query = strings.ToLower(query)
	var matches []Command
	var scores []int

	for _, id := range r.order {
		cmd, ok := r.commands[id]
		if !ok {
			continue
		}

		// Calculate fuzzy match score
		name := strings.ToLower(cmd.Name)
		score := fuzzyScore(query, name)

		if score > 0 {
			matches = append(matches, cmd)
			scores = append(scores, score)
		}
	}

	// Sort by score (descending)
	sort.Slice(matches, func(i, j int) bool {
		return scores[i] > scores[j]
	})

	return matches
}

// fuzzyScore calculates a simple fuzzy match score
func fuzzyScore(pattern, text string) int {
	if len(pattern) == 0 {
		return 1
	}
	if len(text) == 0 {
		return 0
	}

	patternIdx := 0
	score := 0
	lastMatchIdx := -1

	for i := 0; i < len(text) && patternIdx < len(pattern); i++ {
		if text[i] == pattern[patternIdx] {
			score += 10
			// Bonus for consecutive matches
			if lastMatchIdx == i-1 {
				score += 5
			}
			// Bonus for matching at word boundary
			if i == 0 || text[i-1] == ' ' {
				score += 10
			}
			lastMatchIdx = i
			patternIdx++
		}
	}

	// Pattern not fully matched
	if patternIdx < len(pattern) {
		return 0
	}

	return score
}
