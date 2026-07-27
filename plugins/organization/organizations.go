package organization

import (
	"context"

	"github.com/thecodearcher/limen"
)

func (o *organizationPlugin) CreateOrganization(ctx context.Context, user *limen.User, request *CreateOrganizationRequest) (*Organization, error) {
	if o.config.slugGenerator != nil && request.Slug == "" {
		request.Slug = o.config.slugGenerator(request.Name)
	}

	existing, err := o.core.Exists(ctx, o.organizationSchema, []limen.Where{
		limen.Eq(o.organizationSchema.GetSlugField(), request.Slug),
	})

	if err != nil {
		return nil, err
	}

	if existing {
		return nil, ErrOrganizationSlugAlreadyExists
	}

	if o.config.allowOrgCreation != nil {
		if !o.config.allowOrgCreation(ctx, user) {
			return nil, ErrOrganizationCreationNotAllowed
		}
	}

	if err := o.hasUserReachedMaxOrganizations(ctx, user); err != nil {
		return nil, err
	}

	if o.hooks.BeforeCreateOrganization != nil {
		if err := o.hooks.BeforeCreateOrganization(ctx, user, request); err != nil {
			return nil, err
		}
	}

	var organization *Organization
	var owner *Member
	payload := &Organization{
		Name: request.Name,
		Slug: request.Slug,
		Logo: request.Logo,
	}
	if err := o.core.WithTransaction(ctx, func(ctx context.Context) error {
		organizationModel, err := o.core.CreateAndReturn(ctx, o.organizationSchema, payload, request.AdditionalFields, OrganizationSchemaSlugField)
		if err != nil {
			return err
		}
		organization = organizationModel.(*Organization)
		owner, err = o.createOrganizationOwner(ctx, user, organization)
		return err
	}); err != nil {
		return nil, err
	}

	if o.hooks.AfterCreateOrganization != nil {
		o.hooks.AfterCreateOrganization(ctx, organization, user, owner)
	}

	return organization, nil
}

func (o *organizationPlugin) createOrganizationOwner(ctx context.Context, user *limen.User, organization *Organization) (*Member, error) {
	memberModel, err := o.core.CreateAndReturn(ctx, o.memberSchema, &Member{
		OrganizationID: organization.ID,
		UserID:         user.ID,
	}, nil, MemberSchemaOrganizationIDField, MemberSchemaUserIDField)
	if err != nil {
		return nil, err
	}

	ownerRole := o.getOwnerRole()
	if ownerRole == nil {
		return nil, ErrOwnerRoleNotFound
	}

	if err := o.core.Create(ctx, o.memberRoleSchema, &MemberRole{
		MemberID: memberModel.(*Member).ID,
		Role:     ownerRole.Name(),
	}, nil); err != nil {
		return nil, err
	}

	return memberModel.(*Member), nil
}

func (o *organizationPlugin) hasUserReachedMaxOrganizations(ctx context.Context, user *limen.User) error {
	if o.config.maxOrgPerUser == 0 {
		return nil
	}

	count, err := o.core.Count(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetUserIDField(), user.ID),
	})

	if err != nil {
		return err
	}

	if count >= int64(o.config.maxOrgPerUser) {
		return ErrUserHasReachedMaxOrganizations
	}
	return nil
}

func (o *organizationPlugin) ListOrganizations(ctx context.Context, user *limen.User, filter *ListOrganizationsFilter, opts *limen.QueryOptions) (*limen.Page[*Organization], error) {
	memberships, err := o.core.FindMany(ctx, o.memberSchema, []limen.Where{
		limen.Eq(o.memberSchema.GetUserIDField(), user.ID),
	})

	if err != nil {
		return nil, err
	}

	orgIds := make([]any, len(memberships))
	for i, membership := range memberships {
		orgIds[i] = membership.(*Member).OrganizationID
	}

	conditions := []limen.Where{
		limen.In(o.organizationSchema.GetIDField(), orgIds),
	}

	if filter.Name != nil {
		conditions = append(conditions, limen.Contains(o.organizationSchema.GetNameField(), *filter.Name))
	}

	organizations, err := o.core.FindWithOptions(ctx, o.organizationSchema, conditions, opts)
	return limen.MapPage[*Organization](organizations), err
}
