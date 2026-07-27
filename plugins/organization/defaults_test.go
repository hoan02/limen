package organization

import (
	"strings"
	"testing"

	"github.com/thecodearcher/limen/access"
)

func defaultTestRoles(t *testing.T, ac *access.AccessControl) (owner, admin, member access.Role) {
	t.Helper()
	roles, err := DefaultRoles(ac)
	if err != nil {
		t.Fatal(err)
	}
	return roles[0], roles[1], roles[2]
}

func TestDefaultStatementsReturnsCopy(t *testing.T) {
	got := DefaultStatements()

	got["organization"] = nil
	got["injected"] = []string{"boom"}

	fresh := DefaultStatements()
	if _, ok := fresh["injected"]; ok {
		t.Fatal("mutating the returned statements leaked into the defaults")
	}
	if len(fresh["organization"]) == 0 {
		t.Fatal("mutating the returned statements leaked into the defaults")
	}
}

func TestDefaultRoleMatrix(t *testing.T) {
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
			if tt.role.Name() != tt.name {
				t.Fatalf("role name = %q, want %q", tt.role.Name(), tt.name)
			}
			for _, perm := range tt.granted {
				resource, action, _ := strings.Cut(perm, ":")
				if !tt.role.Can(resource, action) {
					t.Errorf("%s should grant %s", tt.name, perm)
				}
			}
			for _, perm := range tt.denied {
				resource, action, _ := strings.Cut(perm, ":")
				if tt.role.Can(resource, action) {
					t.Errorf("%s should not grant %s", tt.name, perm)
				}
			}
		})
	}
}

func TestDefaultRolesAgainstExtendedAC(t *testing.T) {
	stmts := DefaultStatements()
	stmts["organization"] = append(stmts["organization"], "archive")
	ac := access.New(access.MergeStatements(stmts, access.Statements{
		"project": {"create", "read", "delete"},
	}))

	owner, admin, _ := defaultTestRoles(t, ac)

	if !owner.Can("organization", "archive") {
		t.Fatal("owner's organization wildcard should match the extended action")
	}
	if admin.Can("organization", "archive") {
		t.Fatal("admin has no organization wildcard and should not match archive")
	}

	extended, err := DefaultOwnerRole(ac, access.Permissions{"project": {"*"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, perm := range []string{"organization:delete", "member:update", "invitation:cancel", "project:create"} {
		resource, action, _ := strings.Cut(perm, ":")
		if !extended.Can(resource, action) {
			t.Errorf("extended owner should grant %s", perm)
		}
	}
	if owner.Can("project", "create") {
		t.Fatal("extending owner must not affect the base owner role")
	}
}

func TestDefaultRolesRequireVocabulary(t *testing.T) {
	if _, err := DefaultRoles(access.New(access.Statements{"project": {"read"}})); err == nil {
		t.Fatal("building defaults against an AC missing the built-in vocabulary should error")
	}
}
