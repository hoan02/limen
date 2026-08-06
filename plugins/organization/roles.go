package organization

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (o *organizationPlugin) CreateOrganizationRole(ctx context.Context, user *limen.User, organization *Organization, req *CreateOrganizationRoleRequest) (*OrganizationRole, error) {
	actor, err := o.loadMemberAccess(ctx, organization.ID, user.ID)
	if err != nil {
		return nil, err
	}
	if err := actor.requirePermissions(perms("role:create")); err != nil {
		return nil, err
	}

	if err := o.checkCustomRolesEnabled(); err != nil {
		return nil, err
	}

	if err := o.validateOrganizationRoleName(ctx, organization, req.Name); err != nil {
		return nil, err
	}

	if err := o.checkOrganizationRoleLimit(ctx, organization); err != nil {
		return nil, err
	}

	if err := o.validateGrantablePermissions(actor, req.Name, access.Permissions(req.Permissions)); err != nil {
		return nil, err
	}

	if o.hooks.BeforeCreateOrganizationRole != nil {
		if err := o.hooks.BeforeCreateOrganizationRole(ctx, user, organization, req); err != nil {
			return nil, err
		}
	}

	roleModel, err := o.core.CreateAndReturn(ctx, o.organizationRoleSchema, &OrganizationRole{
		OrganizationID: organization.ID,
		Name:           strings.ToLower(strings.TrimSpace(req.Name)),
		Description:    req.Description,
		Permissions:    req.Permissions,
	}, nil, OrganizationRoleSchemaOrganizationIDField, OrganizationRoleSchemaNameField)
	if err != nil {
		return nil, err
	}

	role := roleModel.(*OrganizationRole)
	if o.hooks.AfterCreateOrganizationRole != nil {
		o.hooks.AfterCreateOrganizationRole(ctx, user, organization, role)
	}
	return role, nil
}

func (o *organizationPlugin) UpdateOrganizationRole(ctx context.Context, user *limen.User, organization *Organization, roleID any, req *UpdateOrganizationRoleRequest) (*OrganizationRole, error) {
	actor, err := o.loadMemberAccess(ctx, organization.ID, user.ID)
	if err != nil {
		return nil, err
	}
	if err := actor.requirePermissions(perms("role:update")); err != nil {
		return nil, err
	}

	role, err := o.GetOrganizationRole(ctx, organization, roleID)
	if err != nil {
		return nil, err
	}

	if req.Permissions != nil {
		if err := o.validateGrantablePermissions(actor, role.Name, access.Permissions(req.Permissions)); err != nil {
			return nil, err
		}
	}

	if o.hooks.BeforeUpdateOrganizationRole != nil {
		if err := o.hooks.BeforeUpdateOrganizationRole(ctx, user, organization, role, req); err != nil {
			return nil, err
		}
	}

	payload := make(map[limen.SchemaField]any)

	if req.Permissions != nil {
		encoded, err := json.Marshal(req.Permissions)
		if err != nil {
			return nil, err
		}
		payload[OrganizationRoleSchemaPermissionsField] = string(encoded)
	}

	if req.Description != nil {
		payload[OrganizationRoleSchemaDescriptionField] = *req.Description
	}

	if len(payload) == 0 {
		return role, nil
	}

	updated, err := o.core.UpdateAndReturn(ctx, o.organizationRoleSchema, payload, []limen.Where{
		limen.Eq(o.organizationRoleSchema.GetIDField(), role.ID),
		limen.Eq(o.organizationRoleSchema.GetOrganizationIDField(), organization.ID),
	}, role.ID)
	if err != nil {
		return nil, err
	}

	updatedRole := updated.(*OrganizationRole)
	if o.hooks.AfterUpdateOrganizationRole != nil {
		o.hooks.AfterUpdateOrganizationRole(ctx, user, organization, updatedRole)
	}
	return updatedRole, nil
}

func (o *organizationPlugin) DeleteOrganizationRole(ctx context.Context, user *limen.User, organization *Organization, roleID any) error {
	if err := o.HasPermission(ctx, user, organization.ID, perms("role:delete")); err != nil {
		return err
	}

	role, err := o.GetOrganizationRole(ctx, organization, roleID)
	if err != nil {
		return err
	}

	if err = o.checkOrganizationRoleAssignments(ctx, organization, role); err != nil {
		return err
	}

	if o.hooks.BeforeDeleteOrganizationRole != nil {
		if err := o.hooks.BeforeDeleteOrganizationRole(ctx, user, organization, role); err != nil {
			return err
		}
	}

	if err := o.core.Delete(ctx, o.organizationRoleSchema, []limen.Where{
		limen.Eq(o.organizationRoleSchema.GetIDField(), role.ID),
		limen.Eq(o.organizationRoleSchema.GetOrganizationIDField(), organization.ID),
	}); err != nil {
		return err
	}

	if o.hooks.AfterDeleteOrganizationRole != nil {
		o.hooks.AfterDeleteOrganizationRole(ctx, user, organization, role)
	}
	return nil
}

func (o *organizationPlugin) GetOrganizationRole(ctx context.Context, organization *Organization, roleID any) (*OrganizationRole, error) {
	if err := o.checkCustomRolesEnabled(); err != nil {
		return nil, err
	}

	role, err := o.core.FindOne(ctx, o.organizationRoleSchema, []limen.Where{
		limen.Eq(o.organizationRoleSchema.GetIDField(), roleID),
		limen.Eq(o.organizationRoleSchema.GetOrganizationIDField(), organization.ID),
	}, nil)
	if err != nil {
		return nil, err
	}
	return role.(*OrganizationRole), nil
}

func (o *organizationPlugin) ListOrganizationRoles(ctx context.Context, user *limen.User, organization *Organization, opts *limen.QueryOptions) (*limen.Page[*OrganizationRole], error) {
	if err := o.checkCustomRolesEnabled(); err != nil {
		return nil, err
	}

	if err := o.HasPermission(ctx, user, organization.ID, perms("role:read")); err != nil {
		return nil, err
	}

	roles, err := o.core.FindWithOptions(ctx, o.organizationRoleSchema, []limen.Where{
		limen.Eq(o.organizationRoleSchema.GetOrganizationIDField(), organization.ID),
	}, opts)
	if err != nil {
		return nil, err
	}
	return limen.MapPage[*OrganizationRole](roles), nil
}

func (o *organizationPlugin) checkCustomRolesEnabled() error {
	if !o.config.customRolesEnabled {
		return ErrCustomRolesDisabled
	}
	return nil
}

func (o *organizationPlugin) checkOrganizationRoleLimit(ctx context.Context, organization *Organization) error {
	maxRoles := 0
	switch v := o.config.maxRolesPerOrganization.(type) {
	case int:
		maxRoles = v
	case MaxRolesPerOrganizationFunc:
		maxRoles = v(ctx, organization)
	}

	if maxRoles <= 0 {
		return nil
	}

	count, err := o.core.Count(ctx, o.organizationRoleSchema, []limen.Where{
		limen.Eq(o.organizationRoleSchema.GetOrganizationIDField(), organization.ID),
	})
	if err != nil {
		return err
	}

	if count >= int64(maxRoles) {
		return ErrMaxRolesPerOrganizationReached
	}
	return nil
}

func (o *organizationPlugin) checkOrganizationRoleAssignments(ctx context.Context, organization *Organization, role *OrganizationRole) error {
	assignments, err := o.core.Count(ctx, o.memberRoleSchema, []limen.Where{
		limen.Eq(o.memberRoleSchema.GetOrganizationIDField(), organization.ID),
		limen.Eq(o.memberRoleSchema.GetOrganizationRoleIDField(), role.ID),
	})
	if err != nil {
		return err
	}
	if assignments > 0 {
		return ErrRoleStillAssignedToMembers
	}
	return nil
}

func (o *organizationPlugin) validateOrganizationRoleName(ctx context.Context, organization *Organization, name string) error {
	if name == "" {
		return ErrRoleNameCannotBeEmpty
	}

	if o.isReservedRoleName(name) {
		return ErrRoleNameReserved
	}

	existing, err := o.core.Exists(ctx, o.organizationRoleSchema, []limen.Where{
		limen.Eq(o.organizationRoleSchema.GetOrganizationIDField(), organization.ID),
		limen.Eq(o.organizationRoleSchema.GetNameField(), strings.ToLower(name)),
	})
	if err != nil {
		return err
	}

	if existing {
		return ErrRoleNameAlreadyExists
	}
	return nil
}

func (o *organizationPlugin) isReservedRoleName(name string) bool {
	return slices.ContainsFunc(o.config.roles, func(role access.Role) bool {
		return strings.EqualFold(role.Name(), name)
	})
}

func (o *organizationPlugin) validateGrantablePermissions(actor memberAccess, name string, permissions access.Permissions) error {
	if len(permissions) == 0 {
		return ErrRolePermissionsCannotBeEmpty
	}

	if _, err := o.config.accessControl.NewRole(name, permissions); err != nil {
		return limen.NewLimenError(err.Error(), http.StatusBadRequest, nil)
	}

	return actor.requireGrantable(permissions)
}

// ensureCanGrantRoles verifies the actor may assign/invite the given roles:
// special-role gates (e.g. owner) plus a permission-subset check so actors
// cannot grant permissions they do not hold.
func (o *organizationPlugin) ensureCanGrantRoles(actor memberAccess, roles []*access.Role) error {
	if err := o.ensureCanActOnSpecialRoles(actor, roles); err != nil {
		return err
	}

	for _, role := range roles {
		if err := actor.requireGrantable(role.Permissions()); err != nil {
			return err
		}
	}
	return nil
}

func (o *organizationPlugin) getOrganizationDynamicRoles(ctx context.Context, organization *Organization, roleIDs []any) ([]*access.Role, error) {
	if !o.config.customRolesEnabled {
		return nil, nil
	}

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

func (o *organizationPlugin) composeMemberRoles(memberRoles []*MemberRole, dynamicRolesByID map[any]*access.Role) ([]*access.Role, error) {
	resolved := make([]*access.Role, 0, len(memberRoles))
	for _, memberRole := range memberRoles {
		if memberRole.OrganizationRoleID != nil {
			if role := dynamicRolesByID[memberRole.OrganizationRoleID]; role != nil {
				resolved = append(resolved, role)
			}
			continue
		}

		role, err := o.resolveMemberStaticRole(memberRole)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, role)
	}
	return resolved, nil
}

func (o *organizationPlugin) resolveRoles(ctx context.Context, organization *Organization, roles []any) ([]*access.Role, error) {
	resolvedRoles := make([]*access.Role, 0)
	possiblyDynamicRoleIDs := make([]any, 0)

	if len(roles) == 0 {
		return nil, ErrFailedToResolveRoles
	}

	for _, role := range roles {
		if r, ok := role.(*access.Role); ok {
			resolvedRoles = append(resolvedRoles, r)
			continue
		}

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
		possiblyDynamicRoleIDs = append(possiblyDynamicRoleIDs, o.core.Schema.NormalizeIDValue(role))
	}

	if len(resolvedRoles) == len(roles) {
		return resolvedRoles, nil
	}

	dynamicRoles, err := o.getOrganizationDynamicRoles(ctx, organization, possiblyDynamicRoleIDs)
	if err != nil {
		return nil, err
	}
	resolvedRoles = append(resolvedRoles, dynamicRoles...)

	if len(resolvedRoles) != len(roles) {
		return nil, ErrFailedToResolveRoles
	}
	return resolvedRoles, nil
}

func roleKey(role *access.Role) any {
	if id := role.ID(); id != nil {
		return id
	}
	return role.Name()
}

func (o *organizationPlugin) ensureCanActOnSpecialRoles(actor memberAccess, actedOnRoles []*access.Role) error {
	for _, role := range actedOnRoles {
		if o.isOwnerRole(role) && !o.actorHasOwnerRole(actor) {
			return ErrUserCannotManageOwnerRole
		}
	}
	return nil
}
