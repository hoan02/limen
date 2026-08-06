package organization

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func TestAuthorization_OwnerGates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		seedTarget bool
		wantErr    error
		act        func(t *testing.T, api API, owner, actor *limen.User, org *Organization, targetMemberID any) error
	}{
		{
			name:       "non-owner cannot assign owner",
			seedTarget: true,
			wantErr:    ErrUserCannotManageOwnerRole,
			act: func(t *testing.T, api API, _, actor *limen.User, org *Organization, targetMemberID any) error {
				t.Helper()
				return api.AssignMemberRole(t.Context(), actor, org, targetMemberID, roleNameOwner)
			},
		},
		{
			name:    "non-owner cannot invite owner",
			wantErr: ErrUserCannotInviteOwner,
			act: func(t *testing.T, api API, _, actor *limen.User, org *Organization, _ any) error {
				t.Helper()
				_, err := api.CreateInvitation(t.Context(), actor, org, &CreateInvitationRequest{
					Email: "new-owner@test.com",
					Role:  roleNameOwner,
				})
				return err
			},
		},
		{
			name:    "non-owner cannot remove owner",
			wantErr: ErrUserCannotManageOwnerRole,
			act: func(t *testing.T, api API, owner, actor *limen.User, org *Organization, _ any) error {
				t.Helper()
				return api.RemoveMember(t.Context(), actor, org, memberIDForUser(t, api, org, owner))
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			slug := fmt.Sprintf("owner-gate-%d", i)
			l, api := newTestOrgPlugin(t)
			owner, org, _ := seedOwnerOrg(t, l, api, slug+"-owner@test.com", "Owner Gate", slug)

			actor := seedUser(t, l, slug+"-actor@test.com")
			inviteAndAccept(t, api, owner, org, actor, roleNameAdmin)

			var targetMemberID any
			if tt.seedTarget {
				target := seedUser(t, l, slug+"-target@test.com")
				inviteAndAccept(t, api, owner, org, target, roleNameMember)
				targetMemberID = memberIDForUser(t, api, org, target)
			}

			err := tt.act(t, api, owner, actor, org, targetMemberID)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAuthorization_InsufficientPermissionWiring(t *testing.T) {
	t.Parallel()

	ac := access.New(DefaultStatements())
	ownerRole, err := DefaultOwnerRole(ac)
	require.NoError(t, err)
	adminRole, err := DefaultAdminRole(ac)
	require.NoError(t, err)
	viewer, err := ac.NewRole("viewer", access.Permissions{
		"organization": {"read"},
		"member":       {"read"},
		"invitation":   {"read"},
	})
	require.NoError(t, err)

	l, api := newTestOrgPlugin(t, WithAccessControl(ac), WithRoles(ownerRole, adminRole, viewer))
	owner, org, _ := seedOwnerOrg(t, l, api, "owner-perm-wire@test.com", "Perm Wire", "perm-wire")

	actor := seedUser(t, l, "viewer-perm-wire@test.com")
	inviteAndAccept(t, api, owner, org, actor, "viewer")

	target := seedUser(t, l, "target-perm-wire@test.com")
	inviteAndAccept(t, api, owner, org, target, "viewer")
	targetMemberID := memberIDForUser(t, api, org, target)

	err = api.AssignMemberRole(t.Context(), actor, org, targetMemberID, "viewer")
	assert.ErrorIs(t, err, ErrInsufficientPermission)

	err = api.HasPermission(t.Context(), actor, org.ID, access.P("member:update"))
	assert.ErrorIs(t, err, ErrInsufficientPermission)

	err = api.HasPermission(t.Context(), actor, org.ID, access.P("organization:read"))
	assert.NoError(t, err)
}

func TestCustomRole_PermissionsExceedGranted(t *testing.T) {
	t.Parallel()

	ac := access.New(DefaultStatements())
	ownerRole, err := DefaultOwnerRole(ac)
	require.NoError(t, err)
	adminRole, err := ac.NewRole(roleNameAdmin, access.Permissions{
		"organization": {"read", "update"},
		"member":       {"*"},
		"invitation":   {"*"},
		"role":         {"*"},
	})
	require.NoError(t, err)
	memberRole, err := DefaultMemberRole(ac)
	require.NoError(t, err)

	l, api := newTestOrgPlugin(t,
		WithCustomRoles(true),
		WithAccessControl(ac),
		WithRoles(ownerRole, adminRole, memberRole),
	)
	owner, org, _ := seedOwnerOrg(t, l, api, "owner-custom@test.com", "Custom Roles", "custom-roles")
	adminUser := seedUser(t, l, "admin-custom@test.com")
	inviteAndAccept(t, api, owner, org, adminUser, roleNameAdmin)

	_, err = api.CreateOrganizationRole(t.Context(), adminUser, org, &CreateOrganizationRoleRequest{
		Name: "elevated",
		Permissions: map[string][]string{
			"organization": {"delete"},
		},
	})
	assert.ErrorIs(t, err, ErrRolePermissionsExceedGranted)

	custom, err := api.CreateOrganizationRole(t.Context(), owner, org, &CreateOrganizationRoleRequest{
		Name: "org-deleter",
		Permissions: map[string][]string{
			"organization": {"delete"},
		},
	})
	require.NoError(t, err)

	target := seedUser(t, l, "target-custom@test.com")
	inviteAndAccept(t, api, owner, org, target, roleNameMember)
	targetMemberID := memberIDForUser(t, api, org, target)

	err = api.AssignMemberRole(t.Context(), adminUser, org, targetMemberID, custom.ID)
	assert.ErrorIs(t, err, ErrRolePermissionsExceedGranted)
}
