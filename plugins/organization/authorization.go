package organization

import (
	"context"
	"errors"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

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

func (o *organizationPlugin) HasPermission(ctx context.Context, user *limen.User, organizationID any, permissions access.Permissions) error {
	userPermissions, err := o.GetMemberPermissions(ctx, organizationID, user)
	if err != nil {
		return err
	}

	if !access.HasRequiredPermissions(userPermissions, permissions) {
		return ErrInsufficientPermission
	}
	return nil
}

// GetMemberPermissions returns the union of every permission the user holds in the organization,
// across both configured roles and organization-defined roles.
func (o *organizationPlugin) GetMemberPermissions(ctx context.Context, organizationID any, user *limen.User) (access.Permissions, error) {
	member, err := o.GetMemberByUserID(ctx, organizationID, user.ID)
	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return nil, ErrMemberNotInOrganization
		}
		return nil, err
	}

	roles, err := o.core.FindMany(ctx, o.memberRoleSchema, []limen.Where{
		limen.Eq(o.memberRoleSchema.GetMemberIDField(), member.ID),
	})
	if err != nil {
		return nil, err
	}

	memberRoles := limen.MapToSliceOfType[*MemberRole](roles)

	userPermissions := make(access.Permissions)
	dynamicMemberRoles := make([]*MemberRole, 0)
	for _, memberRole := range memberRoles {
		if memberRole.OrganizationRoleID != nil {
			dynamicMemberRoles = append(dynamicMemberRoles, memberRole)
			continue
		}
		role, err := o.resolveMemberStaticRole(memberRole)
		if err != nil {
			return nil, err
		}
		userPermissions = access.MergePermissions(userPermissions, role.Permissions())
	}

	if len(dynamicMemberRoles) == 0 {
		return userPermissions, nil
	}

	dynamicRoles, err := o.resolveMemberDynamicRoles(ctx, dynamicMemberRoles)
	if err != nil {
		return nil, err
	}

	for _, role := range dynamicRoles {
		userPermissions = access.MergePermissions(userPermissions, role.Permissions())
	}

	return userPermissions, nil
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
