// Package access provides the permission model used by limen plugins and
// applications: a declared vocabulary, roles built against it, and permission
// checks.
package access

import (
	"slices"
)

// Wildcard grants every action on a resource when used in Permissions.
const Wildcard = "*"

// Statements declares a permission vocabulary: resource → allowed actions.
type Statements map[string][]string

// Permissions is a set of grants: resource → granted actions.
type Permissions map[string][]string

// AccessControl holds a permission vocabulary that grants are validated
// against. It is immutable after New and safe for concurrent use.
type AccessControl struct {
	statements Statements
}

// New returns an AccessControl for the given statements.
func New(statements Statements) *AccessControl {
	normalized := make(Statements, len(statements))
	for resource, actions := range statements {
		sorted := slices.Clone(actions)
		slices.Sort(sorted)
		normalized[resource] = sorted
	}
	return &AccessControl{statements: normalized}
}

// Statements returns a copy of the vocabulary.
func (ac *AccessControl) Statements() Statements {
	return MergeStatements(ac.statements)
}

// MergeStatements combines statement sets into a fresh copy; later sets win
// per resource.
func MergeStatements(statements ...Statements) Statements {
	out := make(Statements)
	for _, statement := range statements {
		for resource, actions := range statement {
			out[resource] = slices.Clone(actions)
		}
	}
	return out
}

// MergePermissions unions grant sets per resource into a fresh copy. A "*"
// grant absorbs the other actions for its resource.
func MergePermissions(permissions ...Permissions) Permissions {
	out := make(Permissions)
	for _, permission := range permissions {
		for resource, actions := range permission {
			if slices.Contains(out[resource], Wildcard) {
				continue
			}
			if slices.Contains(actions, Wildcard) {
				out[resource] = []string{Wildcard}
				continue
			}
			out[resource] = append(out[resource], actions...)
		}
	}
	return out
}

// HasRequiredPermissions reports whether subjectPermissions covers every
// grant in requiredPermissions.
func HasRequiredPermissions(subjectPermissions, requiredPermissions Permissions) bool {
	for resource, actions := range requiredPermissions {
		for _, action := range actions {
			if !permits(subjectPermissions, resource, action) {
				return false
			}
		}
	}
	return true
}

// ClampPermissions returns requested permissions limited to those in grantable.
func ClampPermissions(requested, grantable Permissions) Permissions {
	out := make(Permissions, len(requested))
	for resource, actions := range requested {
		if filtered := intersectActions(actions, grantable[resource]); len(filtered) > 0 {
			out[resource] = filtered
		}
	}
	return out
}

func intersectActions(requested, allowed []string) []string {
	var out []string
	for _, action := range requested {
		if slices.Contains(allowed, Wildcard) {
			return requested
		}

		if slices.Contains(allowed, action) {
			out = append(out, action)
		}
	}
	return out
}
