package organization

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOrganization_AssignsOwnerRole(t *testing.T) {
	t.Parallel()

	l, api := newTestOrgPlugin(t)
	owner, org, _ := seedOwnerOrg(t, l, api, "owner-create@test.com", "Acme", "acme-create")

	member, err := api.GetMemberWithRelations(t.Context(), owner, org.ID)
	require.NoError(t, err)
	require.Len(t, member.Roles, 1)
	assert.Equal(t, roleNameOwner, member.Roles[0].Name())
}

func TestCreateOrganization_SlugConflict(t *testing.T) {
	t.Parallel()

	l, api := newTestOrgPlugin(t)
	owner := seedUser(t, l, "owner-slug@test.com")

	_, err := api.CreateOrganization(t.Context(), owner, &CreateOrganizationRequest{
		Name: "First",
		Slug: "shared-slug",
	})
	require.NoError(t, err)

	_, err = api.CreateOrganization(t.Context(), owner, &CreateOrganizationRequest{
		Name: "Second",
		Slug: "shared-slug",
	})
	assert.ErrorIs(t, err, ErrOrganizationSlugAlreadyExists)
}

func TestMaxOrgPerUser_Create(t *testing.T) {
	t.Parallel()

	l, api := newTestOrgPlugin(t, WithMaxOrgPerUser(1))
	owner := seedUser(t, l, "owner-max-create@test.com")

	_, err := api.CreateOrganization(t.Context(), owner, &CreateOrganizationRequest{
		Name: "Only",
		Slug: "only-org",
	})
	require.NoError(t, err)

	_, err = api.CreateOrganization(t.Context(), owner, &CreateOrganizationRequest{
		Name: "Second",
		Slug: "second-org",
	})
	assert.ErrorIs(t, err, ErrUserHasReachedMaxOrganizations)
}

func TestMaxOrgPerUser_InviteAccept(t *testing.T) {
	t.Parallel()

	l, api := newTestOrgPlugin(t, WithMaxOrgPerUser(1))
	owner, org, _ := seedOwnerOrg(t, l, api, "owner-max-invite@test.com", "Cap Org", "cap-org")
	invitee := seedUser(t, l, "invitee-max@test.com")

	_, err := api.CreateOrganization(t.Context(), invitee, &CreateOrganizationRequest{
		Name: "Invitee Org",
		Slug: "invitee-org",
	})
	require.NoError(t, err)

	invitation, err := api.CreateInvitation(t.Context(), owner, org, &CreateInvitationRequest{
		Email: invitee.Email,
		Role:  roleNameMember,
	})
	require.NoError(t, err)

	_, err = api.RespondToInvitation(t.Context(), invitee, invitation.Token, InvitationResponseAccept)
	assert.ErrorIs(t, err, ErrUserHasReachedMaxOrganizations)

	err = api.CheckMemberExistsInOrganization(t.Context(), org.ID, invitee.ID)
	assert.ErrorIs(t, err, ErrMemberNotInOrganization)
}
