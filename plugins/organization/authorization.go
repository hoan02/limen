package organization

import (
	"context"
	"errors"
	"slices"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

// memberAccess is a single load of a member's resolved roles and permission union.
type memberAccess struct {
	roles       []*access.Role
	permissions access.Permissions
}

func (o *organizationPlugin) SwitchOrganization(ctx context.Context, session *limen.Session, organizationIdentifier any) (*Organization, *limen.SessionResult, error) {
	if organizationIdentifier == nil {
		result, err := o.SetActiveOrganization(ctx, session, nil)
		if err != nil {
			return nil, nil, err
		}
		return nil, result, nil
	}

	organization, err := o.GetOrganization(ctx, organizationIdentifier)
	if err != nil {
		return nil, nil, err
	}

	if err := o.CheckMemberExistsInOrganization(ctx, organization.ID, session.UserID); err != nil {
		return nil, nil, err
	}

	result, err := o.SetActiveOrganization(ctx, session, organization)
	if err != nil {
		return nil, nil, err
	}
	return organization, result, nil
}

func (o *organizationPlugin) SetActiveOrganization(ctx context.Context, session *limen.Session, organization *Organization) (*limen.SessionResult, error) {
	var value any
	if organization != nil {
		value = o.sessionOrganizationID(organization)
	}
	return o.core.SessionManager.UpdateSession(ctx, session, map[limen.SchemaField]any{
		SessionSchemaActiveOrganizationIDField: value,
	})
}

// sessionOrganizationID pairs the stored organization ID with its client-safe
// form so client-visible session backends never carry the internal ID.
func (o *organizationPlugin) sessionOrganizationID(organization *Organization) limen.SessionValue {
	value := limen.SessionValue{Internal: organization.ID, Client: organization.ID}
	if encoded, ok := o.core.EncodePublicID(o.organizationSchema, organization); ok {
		value.Client = encoded
	}
	return value
}

func (o *organizationPlugin) GetActiveOrganizationID(ctx context.Context, session *limen.Session) (any, error) {
	value, err := o.core.SessionManager.GetSessionData(ctx, session, SessionSchemaActiveOrganizationIDField)
	if err != nil {
		return nil, err
	}
	return o.core.Schema.NormalizeIDValue(value), nil
}

func (o *organizationPlugin) clearActiveOrganizationFromSessions(ctx context.Context, organizationID any, match map[limen.SchemaField]any) error {
	if match == nil {
		match = make(map[limen.SchemaField]any, 1)
	}
	match[SessionSchemaActiveOrganizationIDField] = organizationID

	return o.core.SessionManager.UpdateSessions(ctx, map[limen.SchemaField]any{
		SessionSchemaActiveOrganizationIDField: nil,
	}, match)
}

func (a memberAccess) requirePermissions(required access.Permissions) error {
	if !access.HasRequiredPermissions(a.permissions, required) {
		return ErrInsufficientPermission
	}
	return nil
}

func (a memberAccess) requireGrantable(required access.Permissions) error {
	if !access.HasRequiredPermissions(a.permissions, required) {
		return ErrRolePermissionsExceedGranted
	}
	return nil
}

func (o *organizationPlugin) actorHasOwnerRole(actor memberAccess) bool {
	return slices.ContainsFunc(actor.roles, o.isOwnerRole)
}

// loadMemberAccess fetches the user's membership roles once and derives permissions from them.
func (o *organizationPlugin) loadMemberAccess(ctx context.Context, organizationID, userID any) (memberAccess, error) {
	member, err := o.GetMemberByUserID(ctx, organizationID, userID)
	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return memberAccess{}, ErrMemberNotInOrganization
		}
		return memberAccess{}, err
	}

	roles, err := o.GetMemberRoles(ctx, member.ID)
	if err != nil {
		return memberAccess{}, err
	}

	permissions := make(access.Permissions)
	for _, role := range roles {
		permissions = access.MergePermissions(permissions, role.Permissions())
	}

	return memberAccess{roles: roles, permissions: permissions}, nil
}

func (o *organizationPlugin) HasPermission(ctx context.Context, user *limen.User, organizationID any, permissions access.Permissions) error {
	actor, err := o.loadMemberAccess(ctx, organizationID, user.ID)
	if err != nil {
		return err
	}

	return actor.requirePermissions(permissions)
}

// GetMemberPermissions returns the union of every permission the user holds in the organization,
// across both configured roles and organization-defined roles.
func (o *organizationPlugin) GetMemberPermissions(ctx context.Context, organizationID any, user *limen.User) (access.Permissions, error) {
	actor, err := o.loadMemberAccess(ctx, organizationID, user.ID)
	if err != nil {
		return nil, err
	}
	return actor.permissions, nil
}

func (o *organizationPlugin) CheckMemberExistsInOrganization(ctx context.Context, organizationID, userID any) error {
	exists, err := o.core.Exists(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetOrganizationIDField(), organizationID),
		limen.Eq(o.memberSchema.GetUserIDField(), userID),
	})

	if err != nil {
		return err
	}

	if !exists {
		return ErrMemberNotInOrganization
	}

	return nil
}
