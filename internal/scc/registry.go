package scc

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Registry holds known SCC codes and aliases.
type Registry struct {
	codes   map[string]bool
	aliases map[string]string // alias → canonical SCC
	frozen  bool
}

// registryJSON is the on-disk format for classification.json.
type registryJSON struct {
	Version int               `json:"version"`
	Frozen  bool              `json:"frozen"`
	Codes   []string          `json:"codes"`
	Aliases map[string]string `json:"aliases,omitempty"`
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		codes:   make(map[string]bool),
		aliases: make(map[string]string),
	}
}

// Add registers an SCC code.
func (r *Registry) Add(code string) {
	r.codes[code] = true
}

// Contains checks if an SCC code is registered.
func (r *Registry) Contains(code string) bool {
	return r.codes[code]
}

// AddAlias registers an alias that resolves to a canonical SCC.
func (r *Registry) AddAlias(alias, canonical string) {
	r.aliases[alias] = canonical
}

// ResolveAlias looks up an alias and returns the canonical SCC.
func (r *Registry) ResolveAlias(alias string) (string, bool) {
	canonical, ok := r.aliases[alias]
	return canonical, ok
}

// Freeze marks the registry as frozen.
func (r *Registry) Freeze() {
	r.frozen = true
}

// IsFrozen returns whether the registry is frozen.
func (r *Registry) IsFrozen() bool {
	return r.frozen
}

// Codes returns all registered SCC codes, sorted.
func (r *Registry) Codes() []string {
	codes := make([]string, 0, len(r.codes))
	for code := range r.codes {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// ValidateAgainstFrozen checks that all codes in the frozen registry
// are present in this registry. Returns an error listing missing codes.
func (r *Registry) ValidateAgainstFrozen(frozen *Registry) error {
	var missing []string
	for code := range frozen.codes {
		if !r.codes[code] {
			missing = append(missing, code)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("frozen SCC codes missing from new registry: %v", missing)
	}
	return nil
}

// Save writes the registry to a JSON file.
func (r *Registry) Save(path string) error {
	data := registryJSON{
		Version: 1,
		Frozen:  r.frozen,
		Codes:   r.Codes(),
	}
	if len(r.aliases) > 0 {
		data.Aliases = r.aliases
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}
	jsonData = append(jsonData, '\n')

	return os.WriteFile(path, jsonData, 0644)
}

// LoadRegistry reads a registry from a JSON file.
func LoadRegistry(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}

	var raw registryJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal registry: %w", err)
	}

	r := NewRegistry()
	r.frozen = raw.Frozen
	for _, code := range raw.Codes {
		r.codes[code] = true
	}
	for alias, canonical := range raw.Aliases {
		r.aliases[alias] = canonical
	}

	return r, nil
}
