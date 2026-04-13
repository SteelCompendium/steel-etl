package content

import "fmt"

// Registry maps @type values to their ContentParser implementations.
type Registry struct {
	parsers map[string]ContentParser
}

// NewRegistry creates a registry pre-loaded with all known parsers.
func NewRegistry() *Registry {
	r := &Registry{parsers: make(map[string]ContentParser)}
	r.Register(&ChapterParser{})
	r.Register(&ClassParser{})
	r.Register(&FeatureGroupParser{})
	r.Register(&FeatureParser{})
	r.Register(&AbilityParser{})
	return r
}

// Register adds a parser to the registry.
func (r *Registry) Register(p ContentParser) {
	r.parsers[p.Type()] = p
}

// Get returns the parser for the given type, or an error if not found.
func (r *Registry) Get(typeName string) (ContentParser, error) {
	p, ok := r.parsers[typeName]
	if !ok {
		return nil, fmt.Errorf("no parser registered for type %q", typeName)
	}
	return p, nil
}

// Has returns whether a parser is registered for the given type.
func (r *Registry) Has(typeName string) bool {
	_, ok := r.parsers[typeName]
	return ok
}
