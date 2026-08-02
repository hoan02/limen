package organization

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen/access"
)

func defaultTestRoles(t *testing.T, ac *access.AccessControl) (owner, admin, member access.Role) {
	t.Helper()
	roles, err := DefaultRoles(ac)
	require.NoError(t, err)
	return roles[0], roles[1], roles[2]
}

func TestDefaultStatementsReturnsCopy(t *testing.T) {
	t.Parallel()

	got := DefaultStatements()
	got["organization"] = nil
	got["injected"] = []string{"boom"}

	fresh := DefaultStatements()
	assert.NotContains(t, fresh, "injected")
	assert.NotEmpty(t, fresh["organization"])
}

func TestDefaultRoleMatrix(t *testing.T) {
	t.Parallel()

	ac := access.New(DefaultStatements())
	owner, admin, member := defaultTestRoles(t, ac)

	tests := []struct {
		role    access.Role
		name    string
		granted []string
		denied  []string
	}{
		{
			role:    owner,
			name:    "owner",
			granted: []string{"organization:delete", "organization:update", "member:delete", "invitation:cancel"},
		},
		{
			role:    admin,
			name:    "admin",
			granted: []string{"organization:read", "organization:update", "member:update", "member:delete", "invitation:create", "invitation:cancel"},
			denied:  []string{"organization:delete", "organization:create"},
		},
		{
			role:    member,
			name:    "member",
			granted: []string{"organization:read", "member:read", "invitation:read"},
			denied:  []string{"organization:update", "organization:delete", "member:update", "member:delete", "invitation:create", "invitation:cancel"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.name, tt.role.Name())
			for _, perm := range tt.granted {
				resource, action, _ := strings.Cut(perm, ":")
				assert.Truef(t, tt.role.Can(resource, action), "%s should grant %s", tt.name, perm)
			}
			for _, perm := range tt.denied {
				resource, action, _ := strings.Cut(perm, ":")
				assert.Falsef(t, tt.role.Can(resource, action), "%s should not grant %s", tt.name, perm)
			}
		})
	}
}

func TestDefaultRolesAgainstExtendedAC(t *testing.T) {
	t.Parallel()

	stmts := DefaultStatements()
	stmts["organization"] = append(stmts["organization"], "archive")
	ac := access.New(access.MergeStatements(stmts, access.Statements{
		"project": {"create", "read", "delete"},
	}))

	owner, admin, _ := defaultTestRoles(t, ac)

	assert.True(t, owner.Can("organization", "archive"))
	assert.False(t, admin.Can("organization", "archive"))

	extended, err := DefaultOwnerRole(ac, access.Permissions{"project": {"*"}})
	require.NoError(t, err)
	for _, perm := range []string{"organization:delete", "member:update", "invitation:cancel", "project:create"} {
		resource, action, _ := strings.Cut(perm, ":")
		assert.Truef(t, extended.Can(resource, action), "extended owner should grant %s", perm)
	}
	assert.False(t, owner.Can("project", "create"))
}

func TestDefaultRolesRequireVocabulary(t *testing.T) {
	t.Parallel()

	_, err := DefaultRoles(access.New(access.Statements{"project": {"read"}}))
	assert.Error(t, err)
}
