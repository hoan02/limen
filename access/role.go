package access

import (
	"errors"
	"fmt"
	"slices"
)

// Role is a named, immutable set of granted permissions.
type Role struct {
	name        string
	permissions Permissions
}

// NewRole builds a role with the grants validated against the AccessControl's
// statements. A "*" action grants every action on its resource, including
// actions declared later.
func (ac *AccessControl) NewRole(name string, perms Permissions) (Role, error) {
	if name == "" {
		return Role{}, errors.New("access: role name must not be empty")
	}

	for resource, actions := range perms {
		declared, ok := ac.statements[resource]
		if !ok {
			return Role{}, fmt.Errorf("access: role %q: unknown resource %q", name, resource)
		}
		for _, action := range actions {
			if action == Wildcard {
				continue
			}
			if !slices.Contains(declared, action) {
				return Role{}, fmt.Errorf("access: role %q: unknown action %q for resource %q", name, action, resource)
			}
		}
	}

	return Role{name: name, permissions: MergePermissions(perms)}, nil
}

// Name returns the role's name.
func (r Role) Name() string {
	return r.name
}

// Can reports whether the role grants action on resource.
func (r Role) Can(resource, action string) bool {
	if resource == "" || action == "" {
		return false
	}
	return permits(r.permissions, resource, action)
}

// CanAll reports whether the role grants every listed permission.
func (r Role) CanAll(permissions Permissions) bool {
	if len(permissions) == 0 {
		return false
	}
	return HasRequiredPermissions(r.permissions, permissions)
}

// Permissions returns a copy of the role's grants.
func (r Role) Permissions() Permissions {
	return MergePermissions(r.permissions)
}

func permits(perms Permissions, resource, action string) bool {
	actions, ok := perms[resource]
	if !ok {
		return false
	}
	if slices.Contains(actions, Wildcard) {
		return true
	}
	return slices.Contains(actions, action)
}
