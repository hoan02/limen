package organization

import (
	"context"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type API interface {
	CreateOrganization(ctx context.Context, user *limen.User, req *CreateOrganizationRequest) (*Organization, error)
	GetOrganization(ctx context.Context, organizationID any) (*Organization, error)
	ListOrganizations(ctx context.Context, user *limen.User, filter *ListOrganizationsFilter, opts *limen.QueryOptions) (*limen.Page[*Organization], error)
	LeaveOrganization(ctx context.Context, user *limen.User, organizationID any) error

	CreateMember(ctx context.Context, user *limen.User, organization *Organization, role any) (*Member, error)
	GetMemberByUserID(ctx context.Context, organizationID, userID any) (*Member, error)
	GetMemberByID(ctx context.Context, organization *Organization, memberID any) (*Member, error)
	GetMemberWithRelations(ctx context.Context, user *limen.User, organizationID any) (*Member, error)
	ListMembers(ctx context.Context, organizationID any, opts *limen.QueryOptions) (*limen.Page[*Member], error)
	ListMembersWithRelations(ctx context.Context, organizationID any, opts *limen.QueryOptions) (*limen.Page[*Member], error)
	GetMemberRoles(ctx context.Context, memberID any) ([]*access.Role, error)
	AssignMemberRole(ctx context.Context, user *limen.User, organization *Organization, memberID any, role any) error
	RevokeMemberRole(ctx context.Context, user *limen.User, organization *Organization, memberID any, role any) error
	RemoveMember(ctx context.Context, user *limen.User, organization *Organization, memberID any) error
	CheckMemberExistsInOrganization(ctx context.Context, organizationID, userID any) error

	CreateInvitation(ctx context.Context, user *limen.User, organization *Organization, req *CreateInvitationRequest) (*Invitation, error)
	FindPendingInvitation(ctx context.Context, options *FindPendingInvitationOptions) (*Invitation, error)
	RespondToInvitation(ctx context.Context, user *limen.User, invitationToken string, response InvitationResponse) (*Invitation, error)
	CancelPendingInvitation(ctx context.Context, user *limen.User, organization *Organization, invitationID any) (*Invitation, error)
	ListInvitations(ctx context.Context, organization *Organization, options *ListInvitationsOptions) (*limen.Page[*Invitation], error)

	// SwitchOrganization sets the active organization after verifying membership.
	// Pass nil to clear the active organization.
	// Prefer this for user-initiated switches; use SetActiveOrganization when membership is already known.
	SwitchOrganization(ctx context.Context, session *limen.Session, organizationIdentifier any) (*Organization, error)
	// SetActiveOrganization sets active_organization_id without a membership check.
	// Pass nil to clear the active organization.
	SetActiveOrganization(ctx context.Context, session *limen.Session, organizationID any) (*limen.Session, error)

	HasPermission(ctx context.Context, user *limen.User, organizationID any, permissions access.Permissions) error
}

// Use returns a type-safe API for the organization plugin.
// Panics if the plugin was not registered in Config.Plugins,
// making it suitable for method chaining.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginOrganization)
}

var _ API = (*organizationPlugin)(nil)
