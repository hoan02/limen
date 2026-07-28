package organization

import (
	"context"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type CreateOrganizationRequest struct {
	Name             string         `json:"name"`
	Slug             string         `json:"slug"`
	Logo             *string        `json:"logo,omitempty"`
	AdditionalFields map[string]any `json:"additional_fields"`
}

type ListOrganizationsFilter struct {
	Name *string `json:"name,omitempty"`
}

type config struct {
	accessControl    *access.AccessControl
	roles            []access.Role
	slugGenerator    func(name string) string
	hooks            Hooks
	ownerRole        string
	maxOrgPerUser    int
	allowOrgCreation func(ctx context.Context, user *limen.User) bool
}

type Hooks struct {
	BeforeCreateOrganization func(ctx context.Context, user *limen.User, request *CreateOrganizationRequest) error
	AfterCreateOrganization  func(ctx context.Context, organization *Organization, user *limen.User, owner *Member)
}

type ConfigOption func(*config)

func WithSlugGenerator(slugGenerator func(name string) string) ConfigOption {
	return func(c *config) {
		c.slugGenerator = slugGenerator
	}
}

func WithHooks(hooks Hooks) ConfigOption {
	return func(c *config) {
		c.hooks = hooks
	}
}

func WithRoles(roles ...access.Role) ConfigOption {
	return func(c *config) {
		c.roles = roles
	}
}

func WithAccessControl(accessControl *access.AccessControl) ConfigOption {
	return func(c *config) {
		c.accessControl = accessControl
	}
}

func WithCreatorRole(ownerRole string) ConfigOption {
	return func(c *config) {
		c.ownerRole = ownerRole
	}
}

// WithMaxOrgPerUser sets the maximum number of organizations a user can be a member of.
// If set to 0, there is no limit.
func WithMaxOrgPerUser(maxOrgPerUser int) ConfigOption {
	return func(c *config) {
		c.maxOrgPerUser = maxOrgPerUser
	}
}

func WithAllowOrgCreation(allowOrgCreation func(ctx context.Context, user *limen.User) bool) ConfigOption {
	return func(c *config) {
		c.allowOrgCreation = allowOrgCreation
	}
}
