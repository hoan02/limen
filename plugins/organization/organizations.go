package organization

import (
	"context"
	"strings"

	"github.com/thecodearcher/limen"
)

func (o *organizationPlugin) CreateOrganization(ctx context.Context, user *limen.User, request *CreateOrganizationRequest) (*Organization, error) {
	request.Slug = o.applySlugNormalization(o.config.slugGenerator(request.Name, request.Slug))
	if request.Slug == "" {
		return nil, ErrInvalidSlug
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

	ownerRoleName := ownerRole.Name()
	if err := o.core.Create(ctx, o.memberRoleSchema, &MemberRole{
		OrganizationID: organization.ID,
		MemberID:       memberModel.(*Member).ID,
		Role:           &ownerRoleName,
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

	if len(memberships) == 0 {
		return limen.EmptyPage[*Organization](opts), nil
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

func (o *organizationPlugin) CheckSlugAvailability(ctx context.Context, slug string) (bool, error) {
	slug = o.applySlugNormalization(slug)
	if slug == "" {
		return false, ErrInvalidSlug
	}

	existing, err := o.core.Exists(ctx, o.organizationSchema, []limen.Where{
		limen.Eq(o.organizationSchema.GetSlugField(), slug),
	})
	if err != nil {
		return false, err
	}
	return !existing, nil
}

func (o *organizationPlugin) applySlugNormalization(slug string) string {
	if o.config.normalizeSlugs {
		return normalizeSlug(slug)
	}
	return strings.TrimSpace(slug)
}

func normalizeSlug(value string) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		isAllowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !isAllowed {
			pendingHyphen = b.Len() > 0
			continue
		}
		if pendingHyphen {
			b.WriteRune('-')
			pendingHyphen = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

func defaultSlugGenerator(name string, providedSlug string) string {
	if slug := strings.TrimSpace(providedSlug); slug != "" {
		return slug
	}
	if slug := normalizeSlug(name); slug != "" {
		return slug
	}
	return strings.ToLower(limen.GenerateRandomString(12, limen.CharSetAlphanumeric))
}

func (o *organizationPlugin) GetOrganization(ctx context.Context, organizationID any) (*Organization, error) {
	organization, err := o.core.FindOne(ctx, o.organizationSchema, []limen.Where{
		limen.Eq(o.organizationSchema.GetIDField(), organizationID),
	}, nil)
	if err != nil {
		return nil, err
	}
	return organization.(*Organization), nil
}
