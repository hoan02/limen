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
	UpdateOrganization(ctx context.Context, user *limen.User, organizationID any, req *UpdateOrganizationRequest) (*Organization, error)
	DeleteOrganization(ctx context.Context, user *limen.User, organizationID any) error
	LeaveOrganization(ctx context.Context, user *limen.User, organizationID any) error
	CheckSlugAvailability(ctx context.Context, slug string) (bool, error)

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

	CreateOrganizationRole(ctx context.Context, user *limen.User, organization *Organization, req *CreateOrganizationRoleRequest) (*OrganizationRole, error)
	GetOrganizationRole(ctx context.Context, organization *Organization, roleID any) (*OrganizationRole, error)
	ListOrganizationRoles(ctx context.Context, organization *Organization, opts *limen.QueryOptions) (*limen.Page[*OrganizationRole], error)
	UpdateOrganizationRole(ctx context.Context, user *limen.User, organization *Organization, roleID any, req *UpdateOrganizationRoleRequest) (*OrganizationRole, error)
	DeleteOrganizationRole(ctx context.Context, user *limen.User, organization *Organization, roleID any) error
	GetMemberPermissions(ctx context.Context, organizationID any, user *limen.User) (access.Permissions, error)

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
	SetActiveOrganization(ctx context.Context, session *limen.Session, organizationID any) error

	HasPermission(ctx context.Context, user *limen.User, organizationID any, permissions access.Permissions) error
}

// Use returns a type-safe API for the organization plugin.
// Panics if the plugin was not registered in Config.Plugins,
// making it suitable for method chaining.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginOrganization)
}

var _ API = (*organizationPlugin)(nil)
