package authz

import "sort"

// ScopeSet is an immutable-by-convention set of credential scopes.
type ScopeSet map[Scope]struct{}

func NewScopeSet(scopes ...Scope) ScopeSet {
	set := make(ScopeSet, len(scopes))
	for _, scope := range scopes {
		set[scope] = struct{}{}
	}
	return set
}

func (set ScopeSet) Has(scope Scope) bool { _, ok := set[scope]; return ok }

func (set ScopeSet) HasAll(required ...Scope) bool {
	for _, scope := range required {
		if !set.Has(scope) {
			return false
		}
	}
	return true
}

func (set ScopeSet) HasAny(required ...Scope) bool {
	for _, scope := range required {
		if set.Has(scope) {
			return true
		}
	}
	return false
}

func (set ScopeSet) Scopes() []Scope {
	scopes := make([]Scope, 0, len(set))
	for scope := range set {
		scopes = append(scopes, scope)
	}
	sort.Slice(scopes, func(i, j int) bool { return scopes[i] < scopes[j] })
	return scopes
}

func (set ScopeSet) Clone() ScopeSet { return NewScopeSet(set.Scopes()...) }
