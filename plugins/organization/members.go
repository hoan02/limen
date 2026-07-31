package organization

import (
	"context"
	"slices"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (o *organizationPlugin) CreateMember(ctx context.Context, user *limen.User, organization *Organization, role any) (*Member, error) {
	existing, err := o.core.Exists(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetOrganizationIDField(), organization.ID),
		limen.Eq(o.memberSchema.GetUserIDField(), user.ID),
	})
	if err != nil {
		return nil, err
	}

	if existing {
		return nil, ErrMemberAlreadyExists
	}

	var member *Member
	resolvedRoles, err := o.resolveRoles(ctx, organization, []any{role})
	if err != nil {
		return nil, err
	}
	err = o.core.WithTransaction(ctx, func(ctx context.Context) error {
		memberModel, err := o.core.CreateAndReturn(ctx, o.memberSchema, &Member{
			OrganizationID: organization.ID,
			UserID:         user.ID,
		}, nil, MemberSchemaOrganizationIDField, MemberSchemaUserIDField)
		if err != nil {
			return err
		}

		member = memberModel.(*Member)
		return o.assignMemberRole(ctx, member, resolvedRoles[0])
	})

	if err != nil {
		return nil, err
	}

	return member, nil
}

func (o *organizationPlugin) AssignMemberRole(ctx context.Context, user *limen.User, organization *Organization, memberID any, roleToAssign any) error {
	if err := o.HasPermission(ctx, user, organization.ID, perms("member:update")); err != nil {
		return err
	}

	member, err := o.GetMemberByID(ctx, organization, memberID)
	if err != nil {
		return err
	}

	if err := o.validateMemberBelongsToOrganization(member, organization); err != nil {
		return err
	}

	resolvedRoles, err := o.resolveRoles(ctx, organization, []any{roleToAssign})
	if err != nil {
		return err
	}
	role := resolvedRoles[0]

	if o.isOwnerRole(role) && !o.userCanAssignOwnerRole(ctx, user, organization) {
		return ErrUserCannotManageOwnerRole
	}

	if o.hooks.BeforeAssignMemberRole != nil {
		if err := o.hooks.BeforeAssignMemberRole(ctx, user, organization, member, role); err != nil {
			return err
		}
	}

	if err := o.assignMemberRole(ctx, member, role); err != nil {
		return err
	}

	if o.hooks.AfterAssignMemberRole != nil {
		o.hooks.AfterAssignMemberRole(ctx, user, organization, member, role)
	}
	return nil
}

func (o *organizationPlugin) RevokeMemberRole(ctx context.Context, user *limen.User, organization *Organization, memberID any, roleToRevoke any) error {
	if err := o.HasPermission(ctx, user, organization.ID, perms("member:update")); err != nil {
		return err
	}

	member, err := o.GetMemberByID(ctx, organization, memberID)
	if err != nil {
		return err
	}

	if err := o.validateMemberBelongsToOrganization(member, organization); err != nil {
		return err
	}

	existingRolesCount, err := o.core.Count(ctx, o.memberRoleSchema, []limen.Where{
		limen.Eq(o.memberRoleSchema.GetMemberIDField(), member.ID),
	})

	if err != nil {
		return err
	}

	if existingRolesCount == 1 {
		return ErrMemberMustHaveAtLeastOneRole
	}

	resolvedRole, err := o.resolveRoles(ctx, organization, []any{roleToRevoke})
	if err != nil {
		return err
	}

	role := resolvedRole[0]
	if err := o.validateOwnerRoleCanBeRemoved(ctx, organization, role); err != nil {
		return err
	}

	if o.isOwnerRole(role) && !o.userCanAssignOwnerRole(ctx, user, organization) {
		return ErrUserCannotManageOwnerRole
	}

	conditions := []limen.Where{
		limen.Eq(o.memberRoleSchema.GetMemberIDField(), member.ID),
	}

	if role.ID() != nil {
		conditions = append(conditions, limen.Eq(o.memberRoleSchema.GetOrganizationRoleIDField(), role.ID()))
	}

	if role.ID() == nil {
		conditions = append(conditions, limen.Eq(o.memberRoleSchema.GetRoleField(), role.Name()))
	}

	if o.hooks.BeforeRevokeMemberRole != nil {
		if err := o.hooks.BeforeRevokeMemberRole(ctx, user, organization, member, role); err != nil {
			return err
		}
	}

	if err := o.core.Delete(ctx, o.memberRoleSchema, conditions); err != nil {
		return err
	}

	if o.hooks.AfterRevokeMemberRole != nil {
		o.hooks.AfterRevokeMemberRole(ctx, user, organization, member, role)
	}
	return nil
}

func (o *organizationPlugin) GetMemberByUserID(ctx context.Context, organizationID, userID any) (*Member, error) {
	member, err := o.core.FindOne(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetOrganizationIDField(), organizationID),
		limen.Eq(o.memberSchema.GetUserIDField(), userID),
	}, nil)
	if err != nil {
		return nil, err
	}

	return member.(*Member), nil
}

func (o *organizationPlugin) GetMemberByID(ctx context.Context, organization *Organization, memberID any) (*Member, error) {
	member, err := o.core.FindOne(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetOrganizationIDField(), organization.ID),
		limen.Eq(o.memberSchema.GetIDField(), memberID),
	}, nil)
	if err != nil {
		return nil, err
	}

	return member.(*Member), nil
}

func (o *organizationPlugin) RemoveMember(ctx context.Context, user *limen.User, organization *Organization, memberID any) error {
	if err := o.HasPermission(ctx, user, organization.ID, perms("member:delete")); err != nil {
		return err
	}

	member, err := o.GetMemberByID(ctx, organization, memberID)
	if err != nil {
		return err
	}

	if err := o.validateMemberBelongsToOrganization(member, organization); err != nil {
		return err
	}

	memberRoles, err := o.GetMemberRoles(ctx, member.ID)
	if err != nil {
		return err
	}

	if err := o.validateOwnerRoleCanBeRemoved(ctx, organization, memberRoles...); err != nil {
		return err
	}

	if o.hooks.BeforeRemoveMember != nil {
		if err := o.hooks.BeforeRemoveMember(ctx, user, organization, member); err != nil {
			return err
		}
	}

	if err := o.deleteMember(ctx, organization, member); err != nil {
		return err
	}

	if o.hooks.AfterRemoveMember != nil {
		o.hooks.AfterRemoveMember(ctx, user, organization, member)
	}
	return nil
}

func (o *organizationPlugin) LeaveOrganization(ctx context.Context, user *limen.User, organizationID any) error {
	organization, err := o.GetOrganization(ctx, organizationID)
	if err != nil {
		return err
	}

	member, err := o.GetMemberByUserID(ctx, organization.ID, user.ID)
	if err != nil {
		return err
	}

	if err := o.validateMemberBelongsToOrganization(member, organization); err != nil {
		return err
	}

	memberRoles, err := o.GetMemberRoles(ctx, member.ID)
	if err != nil {
		return err
	}

	if err := o.validateOwnerRoleCanBeRemoved(ctx, organization, memberRoles...); err != nil {
		return err
	}

	return o.deleteMember(ctx, organization, member)
}

func (o *organizationPlugin) GetMemberRoles(ctx context.Context, memberID any) ([]*access.Role, error) {
	roles, err := o.core.FindMany(ctx, o.memberRoleSchema, []limen.Where{
		limen.Eq(o.memberRoleSchema.GetMemberIDField(), memberID),
	})
	if err != nil {
		return nil, err
	}

	memberRoles := limen.MapToSliceOfType[*MemberRole](roles)
	dynamicRolesByID, err := o.resolveDynamicRolesByID(ctx, memberRoles)
	if err != nil {
		return nil, err
	}
	return o.composeMemberRoles(memberRoles, dynamicRolesByID), nil
}

// Get a complete organization member with all its related entities
func (o *organizationPlugin) GetMemberWithRelations(ctx context.Context, user *limen.User, organizationID any) (*Member, error) {
	organization, err := o.GetOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	member, err := o.GetMemberByUserID(ctx, organization.ID, user.ID)
	if err != nil {
		return nil, err
	}

	roles, err := o.GetMemberRoles(ctx, member.ID)
	if err != nil {
		return nil, err
	}

	member.Roles = roles
	member.User = user
	member.Organization = organization
	return member, nil
}

func (o *organizationPlugin) ListMembers(ctx context.Context, organizationID any, opts *limen.QueryOptions) (*limen.Page[*Member], error) {
	members, err := o.core.FindWithOptions(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetOrganizationIDField(), organizationID),
	}, opts)
	if err != nil {
		return nil, err
	}
	return limen.MapPage[*Member](members), nil
}

func (o *organizationPlugin) ListMembersWithRelations(ctx context.Context, user *limen.User, organizationID any, opts *limen.QueryOptions) (*limen.Page[*Member], error) {
	if err := o.HasPermission(ctx, user, organizationID, perms("member:read")); err != nil {
		return nil, err
	}

	page, err := o.ListMembers(ctx, organizationID, opts)
	if err != nil {
		return nil, err
	}
	if err := o.attachMemberRelations(ctx, page.Items); err != nil {
		return nil, err
	}
	return page, nil
}

func (o *organizationPlugin) attachMemberRelations(ctx context.Context, members []*Member) error {
	if len(members) == 0 {
		return nil
	}
	if err := o.attachMemberRoles(ctx, members); err != nil {
		return err
	}
	return o.attachMemberUsers(ctx, members)
}

func (o *organizationPlugin) attachMemberRoles(ctx context.Context, members []*Member) error {
	memberIDs := make([]any, len(members))
	for i, member := range members {
		memberIDs[i] = member.ID
	}

	roles, err := o.core.FindMany(ctx, o.memberRoleSchema, []limen.Where{
		limen.In(o.memberRoleSchema.GetMemberIDField(), memberIDs),
	})
	if err != nil {
		return err
	}

	memberRoles := limen.MapToSliceOfType[*MemberRole](roles)
	rolesByMember := make(map[any][]*MemberRole, len(members))
	for _, memberRole := range memberRoles {
		rolesByMember[memberRole.MemberID] = append(rolesByMember[memberRole.MemberID], memberRole)
	}

	dynamicRolesByID, err := o.resolveDynamicRolesByID(ctx, memberRoles)
	if err != nil {
		return err
	}

	for _, member := range members {
		member.Roles = o.composeMemberRoles(rolesByMember[member.ID], dynamicRolesByID)
	}
	return nil
}

func (o *organizationPlugin) attachMemberUsers(ctx context.Context, members []*Member) error {
	userIDs := make([]any, len(members))
	for i, member := range members {
		userIDs[i] = member.UserID
	}

	users, err := o.core.FindMany(ctx, o.core.Schema.User, []limen.Where{
		limen.In(o.core.Schema.User.GetIDField(), userIDs),
	})
	if err != nil {
		return err
	}

	usersByID := make(map[any]*limen.User, len(users))
	for _, user := range limen.MapToSliceOfType[*limen.User](users) {
		usersByID[user.ID] = user
	}

	for _, member := range members {
		member.User = usersByID[member.UserID]
	}
	return nil
}

func (o *organizationPlugin) assignMemberRole(ctx context.Context, member *Member, role *access.Role) error {
	conditions := []limen.Where{
		limen.Eq(o.memberRoleSchema.GetMemberIDField(), member.ID),
		limen.Eq(o.memberRoleSchema.GetRoleField(), role.Name()),
	}

	if o.config.customRolesEnabled && role.ID() != nil {
		conditions = append(conditions, limen.Eq(o.memberRoleSchema.GetOrganizationRoleIDField(), role.ID()).Or())
	}

	existing, err := o.core.Exists(ctx, o.memberRoleSchema, conditions)
	if err != nil {
		return err
	}
	if existing {
		return ErrMemberRoleAlreadyExists
	}

	roleName := role.Name()
	if err := o.core.Create(ctx, o.memberRoleSchema, &MemberRole{
		OrganizationID:     member.OrganizationID,
		MemberID:           member.ID,
		Role:               &roleName,
		OrganizationRoleID: role.ID(),
	}, nil); err != nil {
		return err
	}

	return nil
}

func (o *organizationPlugin) validateMemberBelongsToOrganization(member *Member, organization *Organization) error {
	if member.OrganizationID != organization.ID {
		return ErrMemberNotInOrganization
	}
	return nil
}

func (o *organizationPlugin) validateOwnerRoleCanBeRemoved(ctx context.Context, organization *Organization, roles ...*access.Role) error {
	hasOwnerRole := slices.ContainsFunc(roles, o.isOwnerRole)

	if !hasOwnerRole {
		return nil
	}

	ownersCount, err := o.core.Count(ctx, o.memberRoleSchema, []limen.Where{
		limen.Eq(o.memberRoleSchema.GetOrganizationIDField(), organization.ID),
		limen.Eq(o.memberRoleSchema.GetRoleField(), o.getOwnerRole().Name()),
	})
	if err != nil {
		return err
	}

	if ownersCount == 1 {
		return ErrCannotRemoveLastOwner
	}
	return nil
}

func (o *organizationPlugin) deleteMember(ctx context.Context, organization *Organization, member *Member) error {
	return o.core.WithTransaction(ctx, func(ctx context.Context) error {
		if err := o.core.Delete(ctx, o.memberRoleSchema, []limen.Where{
			limen.Eq(o.memberRoleSchema.GetMemberIDField(), member.ID),
			limen.Eq(o.memberRoleSchema.GetOrganizationIDField(), organization.ID),
		}); err != nil {
			return err
		}

		if err := o.core.Delete(ctx, o.memberSchema, []limen.Where{
			limen.Eq(o.memberSchema.GetOrganizationIDField(), organization.ID),
			limen.Eq(o.memberSchema.GetIDField(), member.ID),
		}); err != nil {
			return err
		}

		return o.clearActiveOrganizationFromSessions(ctx, organization.ID,
			limen.Eq(o.sessionSchema.GetUserIDField(), member.UserID),
		)
	})
}
