package access

import (
	"context"
	"slices"
)

type Permissions map[string][]string
type Roles map[string]Permissions

type AccessControl struct {
	Permissions Permissions
	Roles       Roles
}

type Statement struct {
	Subject string
	Action  string
}

func NewAccessControl() *AccessControl {
	return &AccessControl{
		Permissions: make(Permissions),
		Roles:       make(Roles),
	}
}

func (a *AccessControl) Check(ctx context.Context, subject, action string) error {
	return nil
}

func (a *AccessControl) WithPermissions(permissions Permissions) *AccessControl {
	a.Permissions = permissions
	return a
}

func (a *AccessControl) HasPermission(subject, action string) bool {
	permissions, ok := a.Permissions[subject]
	if !ok {
		return false
	}
	return slices.Contains(permissions, action)
}

func HasRequiredPermissions(subjectPermissions, requiredPermissions Permissions) bool {
	for resource, actions := range requiredPermissions {
		for _, action := range actions {
			subjectActions, ok := subjectPermissions[resource]
			if !ok {
				return false
			}

			if slices.Contains(subjectActions, "*") {
				continue
			}

			if !slices.Contains(subjectActions, action) {
				return false
			}
		}
	}
	return true
}

func intersectActions(requested, allowed []string) []string {
	var out []string
	for _, action := range requested {
		if slices.Contains(allowed, "*") {
			return requested
		}

		if slices.Contains(allowed, action) {
			out = append(out, action)
		}
	}
	return out
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
