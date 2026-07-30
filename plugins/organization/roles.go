package organization

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (o *organizationPlugin) GetOrganizationDynamicRoles(ctx context.Context, organization *Organization, roleIDs []any) ([]*access.Role, error) {
	conditions := []limen.Where{
		limen.Eq(o.organizationRoleSchema.GetOrganizationIDField(), organization.ID),
	}
	if len(roleIDs) > 0 {
		conditions = append(conditions, limen.In(o.organizationRoleSchema.GetIDField(), roleIDs))
	}
	organizationRoles, err := o.core.FindMany(ctx, o.organizationRoleSchema, conditions)

	if err != nil {
		return nil, err
	}

	dynamicRoles := make([]*access.Role, 0, len(organizationRoles))
	for _, model := range organizationRoles {
		orgRole := model.(*OrganizationRole)
		role, err := o.config.accessControl.NewRoleWithID(orgRole.ID, orgRole.Name, orgRole.Permissions)
		if err != nil {
			return nil, err
		}
		dynamicRoles = append(dynamicRoles, &role)
	}

	return dynamicRoles, nil
}

func (o *organizationPlugin) resolveMemberStaticRole(memberRole *MemberRole) (*access.Role, error) {
	for _, role := range o.config.roles {
		if memberRole.Role != nil && role.Name() == *memberRole.Role {
			return &role, nil
		}
	}
	return nil, limen.NewLimenError(fmt.Sprintf("Role %v not found", memberRole.Role), http.StatusNotFound, nil)
}

func (o *organizationPlugin) resolveDynamicRolesByID(ctx context.Context, memberRoles []*MemberRole) (map[any]*access.Role, error) {
	roleIDs := make([]any, 0)
	seen := make(map[any]struct{})
	for _, memberRole := range memberRoles {
		if memberRole.OrganizationRoleID == nil {
			continue
		}
		if _, ok := seen[memberRole.OrganizationRoleID]; ok {
			continue
		}
		seen[memberRole.OrganizationRoleID] = struct{}{}
		roleIDs = append(roleIDs, memberRole.OrganizationRoleID)
	}

	if len(roleIDs) == 0 {
		return nil, nil
	}

	organizationRoles, err := o.core.FindMany(ctx, o.organizationRoleSchema, []limen.Where{
		limen.In(o.organizationRoleSchema.GetIDField(), roleIDs),
	})
	if err != nil {
		return nil, err
	}

	dynamicRolesByID := make(map[any]*access.Role, len(organizationRoles))
	for _, model := range organizationRoles {
		orgRole := model.(*OrganizationRole)
		role, err := o.config.accessControl.NewRoleWithID(orgRole.ID, orgRole.Name, orgRole.Permissions)
		if err != nil {
			return nil, err
		}
		dynamicRolesByID[orgRole.ID] = &role
	}
	return dynamicRolesByID, nil
}

func (o *organizationPlugin) resolveMemberDynamicRoles(ctx context.Context, memberRoles []*MemberRole) ([]*access.Role, error) {
	dynamicRolesByID, err := o.resolveDynamicRolesByID(ctx, memberRoles)
	if err != nil {
		return nil, err
	}
	dynamicRoles := make([]*access.Role, 0, len(dynamicRolesByID))
	for _, role := range dynamicRolesByID {
		dynamicRoles = append(dynamicRoles, role)
	}
	return dynamicRoles, nil
}

func (o *organizationPlugin) composeMemberRoles(memberRoles []*MemberRole, dynamicRolesByID map[any]*access.Role) []*access.Role {
	var resolved []*access.Role
	for _, memberRole := range memberRoles {
		if memberRole.OrganizationRoleID != nil && dynamicRolesByID[memberRole.OrganizationRoleID] != nil {
			resolved = append(resolved, dynamicRolesByID[memberRole.OrganizationRoleID])
			continue
		}

		if role, err := o.resolveMemberStaticRole(memberRole); err == nil {
			resolved = append(resolved, role)
		}
	}
	return resolved
}

func (o *organizationPlugin) resolveRoles(ctx context.Context, organization *Organization, roles []any) ([]*access.Role, error) {
	resolvedRoles := make([]*access.Role, 0)

	possiblyDynamicRoleIDs := make([]any, 0)

	for _, role := range roles {
		if name, ok := role.(string); ok {
			if i := slices.IndexFunc(o.config.roles, func(r access.Role) bool {
				return r.Name() == name
			}); i != -1 {
				resolvedRoles = append(resolvedRoles, &o.config.roles[i])
				continue
			}

			if o.core.IsPublicID(o.organizationRoleSchema, name) {
				possiblyDynamicRoleIDs = append(possiblyDynamicRoleIDs, name)
				continue
			}
		}

		if !o.core.Schema.MatchesIDColumnType(role) {
			return nil, ErrFailedToResolveRoles
		}
		possiblyDynamicRoleIDs = append(possiblyDynamicRoleIDs, role)
	}

	if len(resolvedRoles) == len(roles) {
		return resolvedRoles, nil
	}

	dynamicRoles, err := o.GetOrganizationDynamicRoles(ctx, organization, possiblyDynamicRoleIDs)
	if err != nil {
		return nil, err
	}
	resolvedRoles = append(resolvedRoles, dynamicRoles...)

	if len(resolvedRoles) != len(roles) {
		return nil, ErrFailedToResolveRoles
	}
	return resolvedRoles, nil
}
