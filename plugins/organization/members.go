package organization

import (
	"context"
	"fmt"
	"net/http"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (o *organizationPlugin) CreateMember(ctx context.Context, user *limen.User, organization *Organization, role string) (*Member, error) {
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
	roleObj, err := o.resolveRole(ctx, role)
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
		return o.AssignMemberRole(ctx, member, roleObj)
	})

	if err != nil {
		return nil, err
	}

	return member, nil
}

func (o *organizationPlugin) AssignMemberRole(ctx context.Context, member *Member, role *access.Role) error {
	existing, err := o.core.Exists(ctx, o.memberRoleSchema, []limen.Where{
		limen.Eq(o.memberRoleSchema.GetMemberIDField(), member.ID),
		limen.Eq(o.memberRoleSchema.GetRoleField(), role.Name()),
	})

	if err != nil {
		return err
	}
	if existing {
		return ErrMemberRoleAlreadyExists
	}

	roleName := role.Name()
	return o.core.Create(ctx, o.memberRoleSchema, &MemberRole{
		MemberID: member.ID,
		Role:     &roleName,
	}, nil)
}

func (o *organizationPlugin) GetMember(ctx context.Context, organizationID, userID any) (*Member, error) {
	member, err := o.core.FindOne(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetOrganizationIDField(), organizationID),
		limen.Eq(o.memberSchema.GetUserIDField(), userID),
	}, nil)
	if err != nil {
		return nil, err
	}

	return member.(*Member), nil
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
	member, err := o.GetMember(ctx, organizationID, user.ID)
	if err != nil {
		return nil, err
	}

	roles, err := o.GetMemberRoles(ctx, member.ID)
	if err != nil {
		return nil, err
	}

	organization, err := o.GetOrganization(ctx, organizationID)
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

func (o *organizationPlugin) ListMembersWithRelations(ctx context.Context, organizationID any, opts *limen.QueryOptions) (*limen.Page[*Member], error) {
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

func (o *organizationPlugin) resolveRole(ctx context.Context, role string) (*access.Role, error) {
	for _, r := range o.config.roles {
		if r.Name() == role {
			return &r, nil
		}
	}
	return nil, limen.NewLimenError(fmt.Sprintf("Role %s not found", role), http.StatusNotFound, nil)
}
