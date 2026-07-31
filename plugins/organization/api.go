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

	CreateMember(ctx context.Context, user *limen.User, organization *Organization, role any) (*Member, error)
	// GetMemberByUserID gets a member by organization ID and user ID.
	GetMemberByUserID(ctx context.Context, organizationID, userID any) (*Member, error)
	// GetMemberInOrganization gets a member by organization ID and member ID.
	GetMemberByID(ctx context.Context, organization *Organization, memberID any) (*Member, error)
	GetMemberWithRelations(ctx context.Context, user *limen.User, organizationID any) (*Member, error)
	ListMembers(ctx context.Context, organizationID any, opts *limen.QueryOptions) (*limen.Page[*Member], error)
	ListMembersWithRelations(ctx context.Context, organizationID any, opts *limen.QueryOptions) (*limen.Page[*Member], error)
	GetMemberRoles(ctx context.Context, memberID any) ([]*access.Role, error)
	AssignMemberRole(ctx context.Context, user *limen.User, organization *Organization, memberID any, role any) error
	RevokeMemberRole(ctx context.Context, user *limen.User, organization *Organization, memberID any, role any) error
	CheckMemberExistsInOrganization(ctx context.Context, organizationID, userID any) error

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
