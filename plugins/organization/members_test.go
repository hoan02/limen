package organization

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestAssignAndRevokeMemberRole(t *testing.T) {
	t.Parallel()

	l, api := newTestOrgPlugin(t)
	owner, org, _ := seedOwnerOrg(t, l, api, "owner-roles@test.com", "Roles Org", "roles-org")
	memberUser := seedUser(t, l, "member-roles@test.com")
	inviteAndAccept(t, api, owner, org, memberUser, roleNameMember)
	memberID := memberIDForUser(t, api, org, memberUser)

	err := api.AssignMemberRole(t.Context(), owner, org, memberID, roleNameAdmin)
	require.NoError(t, err)
	assert.Equal(t, []string{roleNameAdmin, roleNameMember}, memberRoleNames(t, api, memberID))

	err = api.RevokeMemberRole(t.Context(), owner, org, memberID, roleNameAdmin)
	require.NoError(t, err)
	assert.Equal(t, []string{roleNameMember}, memberRoleNames(t, api, memberID))

	err = api.RevokeMemberRole(t.Context(), owner, org, memberID, roleNameMember)
	assert.ErrorIs(t, err, ErrMemberMustHaveAtLeastOneRole)
}

func TestLastOwnerProtection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		act  func(t *testing.T, api API, owner *limen.User, org *Organization, session *limen.Session) error
	}{
		{
			name: "remove",
			act: func(t *testing.T, api API, owner *limen.User, org *Organization, _ *limen.Session) error {
				t.Helper()
				return api.RemoveMember(t.Context(), owner, org, memberIDForUser(t, api, org, owner))
			},
		},
		{
			name: "leave",
			act: func(t *testing.T, api API, _ *limen.User, org *Organization, session *limen.Session) error {
				t.Helper()
				_, err := api.LeaveOrganization(t.Context(), session, org.ID)
				return err
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			slug := fmt.Sprintf("last-owner-%d", i)
			l, api := newTestOrgPlugin(t)
			owner, org, session := seedOwnerOrg(t, l, api, slug+"@test.com", "Last Owner", slug)

			err := tt.act(t, api, owner, org, session)
			assert.ErrorIs(t, err, ErrCannotRemoveLastOwner)
		})
	}
}

func TestLeaveOrganization_ClearsActiveOrganization(t *testing.T) {
	t.Parallel()

	l, api := newTestOrgPlugin(t)
	owner, org, session := seedOwnerOrg(t, l, api, "owner-leave-clear@test.com", "Leave Clear", "leave-clear")
	coOwner := seedUser(t, l, "co-owner-leave-clear@test.com")
	inviteAndAccept(t, api, owner, org, coOwner, roleNameOwner)

	_, err := api.SetActiveOrganization(t.Context(), session, org)
	require.NoError(t, err)
	activeID, err := api.GetActiveOrganizationID(t.Context(), session)
	require.NoError(t, err)
	require.Equal(t, org.ID, activeID)

	_, err = api.LeaveOrganization(t.Context(), session, org.ID)
	require.NoError(t, err)

	activeID, err = api.GetActiveOrganizationID(t.Context(), session)
	require.NoError(t, err)
	assert.Nil(t, activeID)

	err = api.CheckMemberExistsInOrganization(t.Context(), org.ID, owner.ID)
	assert.ErrorIs(t, err, ErrMemberNotInOrganization)
}

func TestLeaveOrganization_DifferentActiveOrgUnchanged(t *testing.T) {
	t.Parallel()

	l, api := newTestOrgPlugin(t)
	owner, orgA, session := seedOwnerOrg(t, l, api, "owner-leave-other@test.com", "Org A", "leave-org-a")
	coOwner := seedUser(t, l, "co-owner-leave-other@test.com")
	inviteAndAccept(t, api, owner, orgA, coOwner, roleNameOwner)

	orgB, err := api.CreateOrganization(t.Context(), owner, &CreateOrganizationRequest{
		Name: "Org B",
		Slug: "leave-org-b",
	})
	require.NoError(t, err)

	_, err = api.SetActiveOrganization(t.Context(), session, orgB)
	require.NoError(t, err)

	result, err := api.LeaveOrganization(t.Context(), session, orgA.ID)
	require.NoError(t, err)
	assert.Nil(t, result)

	activeID, err := api.GetActiveOrganizationID(t.Context(), session)
	require.NoError(t, err)
	assert.Equal(t, orgB.ID, activeID)
}
