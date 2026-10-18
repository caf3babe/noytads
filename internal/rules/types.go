package rules

// Phase 2: Filter list rule types.
// These types will be populated by the parser and consumed by the proxy handler.

// RuleType identifies the category of a filter rule.
type RuleType int

const (
	RuleNetwork  RuleType = iota // ||domain^ — block/allow network requests
	RuleCosmetic                 // domain##selector — hide DOM elements via CSS
	RuleScriptlet                // domain##+js(name, args) — inject JavaScript
)

// Rule represents a single parsed filter list rule.
type Rule struct {
	Type      RuleType
	Raw       string   // Original rule text for debugging
	Domains   []string // Domains this rule applies to (empty = all)
	Exception bool     // true for @@|| (allow) or #@# (unhide) rules

	// Network rule fields
	Pattern string // URL pattern to match

	// Cosmetic rule fields
	Selector string // CSS selector to hide

	// Scriptlet rule fields
	ScriptletName string   // e.g. "json-prune", "set-constant"
	ScriptletArgs []string // Arguments passed to the scriptlet
}

// Index provides O(1) lookup of rules by domain.
type Index struct {
	// domain → rules mapping
	byDomain map[string][]*Rule
	// Rules that apply to all domains (no domain restriction)
	global []*Rule
}

// NewIndex creates an empty rule index.
func NewIndex() *Index {
	return &Index{
		byDomain: make(map[string][]*Rule),
	}
}

// Add inserts a rule into the index.
func (idx *Index) Add(r *Rule) {
	if len(r.Domains) == 0 {
		idx.global = append(idx.global, r)
		return
	}
	for _, d := range r.Domains {
		idx.byDomain[d] = append(idx.byDomain[d], r)
	}
}

// Lookup returns all rules matching the given domain.
func (idx *Index) Lookup(domain string) []*Rule {
	var result []*Rule
	result = append(result, idx.global...)
	result = append(result, idx.byDomain[domain]...)

	// Also check parent domains (e.g. "www.youtube.com" → "youtube.com")
	for i := 0; i < len(domain); i++ {
		if domain[i] == '.' {
			parent := domain[i+1:]
			result = append(result, idx.byDomain[parent]...)
		}
	}

	return result
}
