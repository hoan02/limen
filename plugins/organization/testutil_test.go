package organization

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func newTestOrgPlugin(t *testing.T, opts ...ConfigOption) (*limen.Limen, API) {
	t.Helper()

	ac := access.New(DefaultStatements())
	roles, err := DefaultRoles(ac)
	require.NoError(t, err)

	base := []ConfigOption{
		WithAccessControl(ac),
		WithRoles(roles...),
	}
	plugin := New(append(base, opts...)...)
	l, _ := limen.NewTestLimen(t, plugin)
	return l, Use(l)
}

func seedUser(t *testing.T, l *limen.Limen, email string) *limen.User {
	t.Helper()
	return limen.SeedTestUser(t, l, email)
}

func seedSession(t *testing.T, l *limen.Limen, user *limen.User) *limen.Session {
	t.Helper()
	return limen.SeedTestSessionRecord(t, l, user.ID, user.Email)
}

func seedOwnerOrg(t *testing.T, l *limen.Limen, api API, ownerEmail, name, slug string) (*limen.User, *Organization, *limen.Session) {
	t.Helper()

	owner := seedUser(t, l, ownerEmail)
	org, err := api.CreateOrganization(t.Context(), owner, &CreateOrganizationRequest{
		Name: name,
		Slug: slug,
	})
	require.NoError(t, err)
	require.NotNil(t, org)

	session := seedSession(t, l, owner)
	return owner, org, session
}

func inviteAndAccept(t *testing.T, api API, inviter *limen.User, org *Organization, invitee *limen.User, role any) *Invitation {
	t.Helper()

	invitation, err := api.CreateInvitation(t.Context(), inviter, org, &CreateInvitationRequest{
		Email: invitee.Email,
		Role:  role,
	})
	require.NoError(t, err)
	require.NotEmpty(t, invitation.Token)

	accepted, err := api.RespondToInvitation(t.Context(), invitee, invitation.Token, InvitationResponseAccept)
	require.NoError(t, err)
	require.Equal(t, InvitationStatusAccepted, accepted.Status)
	return accepted
}

func memberIDForUser(t *testing.T, api API, org *Organization, user *limen.User) any {
	t.Helper()
	member, err := api.GetMemberByUserID(t.Context(), org.ID, user.ID)
	require.NoError(t, err)
	return member.ID
}

func memberRoleNames(t *testing.T, api API, memberID any) []string {
	t.Helper()
	roles, err := api.GetMemberRoles(t.Context(), memberID)
	require.NoError(t, err)
	return SortedRoleNames(roles)
}
