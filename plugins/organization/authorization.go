package organization

import (
	"context"
	"errors"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

// SwitchOrganization sets the session's active organization after verifying membership.
// Use this for user-initiated org switches; when membership is already known, use SetActiveOrganization instead.
func (o *organizationPlugin) SwitchOrganization(ctx context.Context, session *limen.Session, organizationIdentifier any) (*Organization, error) {
	organization, err := o.GetOrganization(ctx, organizationIdentifier)
	if err != nil {
		return nil, err
	}

	if err := o.CheckMemberExistsInOrganization(ctx, organization.ID, session.UserID); err != nil {
		return nil, err
	}

	if _, err := o.SetActiveOrganization(ctx, session, organization.ID); err != nil {
		return nil, err
	}
	return organization, nil
}

// SetActiveOrganization sets active_organization_id on the session without checking membership.
// Use when membership is already guaranteed, for example, right after creating an organization.
// For user-initiated switches, use SwitchOrganization instead.
func (o *organizationPlugin) SetActiveOrganization(ctx context.Context, session *limen.Session, organizationID any) (*limen.Session, error) {
	metadata := session.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata[MetadataActiveOrganizationID] = organizationID
	session.Metadata = metadata
	if err := o.core.DBAction.UpdateSession(ctx, map[limen.SchemaField]any{limen.SessionSchemaMetadataField: metadata}, []limen.Where{
		limen.Eq(o.sessionSchema.GetIDField(), session.ID),
	}); err != nil {
		return nil, err
	}
	return session, nil
}

func (o *organizationPlugin) HasPermission(ctx context.Context, user *limen.User, organizationID any, permissions access.Permissions) error {
	member, err := o.GetMember(ctx, organizationID, user.ID)
	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return ErrMemberNotInOrganization
		}
		return err
	}

	roles, err := o.core.FindMany(ctx, o.memberRoleSchema, []limen.Where{
		limen.Eq(o.memberRoleSchema.GetMemberIDField(), member.ID),
	})
	if err != nil {
		return err
	}

	memberRoles := limen.MapToSliceOfType[*MemberRole](roles)

	userPermissions := make(access.Permissions)
	dynamicRoleIDs := make([]*MemberRole, 0)
	for _, memberRole := range memberRoles {
		if memberRole.OrganizationRoleID != nil {
			dynamicRoleIDs = append(dynamicRoleIDs, memberRole)
			continue
		}
		role, err := o.resolveMemberStaticRole(memberRole)
		if err != nil {
			return err
		}
		userPermissions = access.MergePermissions(userPermissions, role.Permissions())
	}

	if access.HasRequiredPermissions(userPermissions, permissions) {
		return nil
	}

	if len(dynamicRoleIDs) == 0 {
		return ErrInsufficientPermission
	}

	dynamicRoles, err := o.resolveMemberDynamicRoles(ctx, dynamicRoleIDs)
	if err != nil {
		return err
	}

	for _, role := range dynamicRoles {
		userPermissions = access.MergePermissions(userPermissions, role.Permissions())
	}

	if access.HasRequiredPermissions(userPermissions, permissions) {
		return nil
	}

	return ErrInsufficientPermission
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
