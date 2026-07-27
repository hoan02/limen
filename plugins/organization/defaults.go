package organization

import (
	"github.com/thecodearcher/limen/access"
)

const (
	roleNameOwner  = "owner"
	roleNameAdmin  = "admin"
	roleNameMember = "member"
)

var defaultStatements = access.Statements{
	"organization": {"create", "read", "update", "delete"},
	"member":       {"read", "update", "delete"},
	"invitation":   {"create", "read", "cancel"},
}

// DefaultStatements returns a copy of the plugin's built-in vocabulary.
func DefaultStatements() access.Statements {
	return access.MergeStatements(defaultStatements)
}

// DefaultOwnerRole builds the built-in owner role against ac.
func DefaultOwnerRole(ac *access.AccessControl, extra ...access.Permissions) (access.Role, error) {
	return ac.NewRole(roleNameOwner, access.MergePermissions(append([]access.Permissions{{
		"organization": {"*"},
		"member":       {"*"},
		"invitation":   {"*"},
	}}, extra...)...))
}

// DefaultAdminRole builds the built-in admin role against ac.
func DefaultAdminRole(ac *access.AccessControl, extra ...access.Permissions) (access.Role, error) {
	return ac.NewRole(roleNameAdmin, access.MergePermissions(append([]access.Permissions{{
		"organization": {"read", "update"},
		"member":       {"*"},
		"invitation":   {"*"},
	}}, extra...)...))
}

// DefaultMemberRole builds the built-in member role against ac.
func DefaultMemberRole(ac *access.AccessControl, extra ...access.Permissions) (access.Role, error) {
	return ac.NewRole(roleNameMember, access.MergePermissions(append([]access.Permissions{{
		"organization": {"read"},
		"member":       {"read"},
		"invitation":   {"read"},
	}}, extra...)...))
}

// DefaultRoles builds the built-in roles against ac.
func DefaultRoles(ac *access.AccessControl) ([]access.Role, error) {
	builders := []func(*access.AccessControl, ...access.Permissions) (access.Role, error){
		DefaultOwnerRole, DefaultAdminRole, DefaultMemberRole,
	}

	roles := make([]access.Role, 0, len(builders))

	for _, builder := range builders {
		role, err := builder(ac)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}
