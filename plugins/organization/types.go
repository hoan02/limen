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

type SendInvitationMailData struct {
	Inviter      *limen.User
	Organization *Organization
	Invitation   *Invitation
}

type MaxMembersPerOrganizationFunc func(ctx context.Context, organization *Organization) int

type MaxRolesPerOrganizationFunc func(ctx context.Context, organization *Organization) int

type SlugGeneratorFunc func(name string, providedSlug string) string

type CreateOrganizationRoleRequest struct {
	Name        string              `json:"name"`
	Description *string             `json:"description,omitempty"`
	Permissions map[string][]string `json:"permissions"`
}

type UpdateOrganizationRoleRequest struct {
	Description *string             `json:"description,omitempty"`
	Permissions map[string][]string `json:"permissions,omitempty"`
}

type config struct {
	accessControl                  *access.AccessControl
	roles                          []access.Role
	slugGenerator                  SlugGeneratorFunc
	normalizeSlugs                 bool
	hooks                          Hooks
	ownerRole                      string
	maxOrgPerUser                  int
	maxMembersPerOrganization      any
	allowOrgCreation               func(ctx context.Context, user *limen.User) bool
	sendInvitationMail             func(ctx context.Context, data *SendInvitationMailData)
	cancelPendingInviteOnNewInvite bool
	invitationExpirationSeconds    int
	customRolesEnabled             bool
	maxRolesPerOrganization        any
}

type Hooks struct {
	BeforeCreateOrganization func(ctx context.Context, user *limen.User, request *CreateOrganizationRequest) error
	AfterCreateOrganization  func(ctx context.Context, organization *Organization, user *limen.User, owner *Member)

	BeforeCreateInvitation func(ctx context.Context, user *limen.User, organization *Organization, request *CreateInvitationRequest) error
	AfterCreateInvitation  func(ctx context.Context, invitation *Invitation, user *limen.User, organization *Organization)

	BeforeCancelInvitation func(ctx context.Context, user *limen.User, organization *Organization, invitation *Invitation) error
	AfterCancelInvitation  func(ctx context.Context, user *limen.User, organization *Organization, invitation *Invitation)

	BeforeRespondToInvitation func(ctx context.Context, user *limen.User, organization *Organization, invitation *Invitation, response InvitationResponse) error
	AfterRespondToInvitation  func(ctx context.Context, user *limen.User, organization *Organization, invitation *Invitation, response InvitationResponse)

	BeforeAssignMemberRole func(ctx context.Context, user *limen.User, organization *Organization, member *Member, role *access.Role) error
	AfterAssignMemberRole  func(ctx context.Context, user *limen.User, organization *Organization, member *Member, role *access.Role)

	BeforeRevokeMemberRole func(ctx context.Context, user *limen.User, organization *Organization, member *Member, role *access.Role) error
	AfterRevokeMemberRole  func(ctx context.Context, user *limen.User, organization *Organization, member *Member, role *access.Role)

	BeforeRemoveMember func(ctx context.Context, user *limen.User, organization *Organization, member *Member) error
	AfterRemoveMember  func(ctx context.Context, user *limen.User, organization *Organization, member *Member) error

	BeforeCreateOrganizationRole func(ctx context.Context, user *limen.User, organization *Organization, request *CreateOrganizationRoleRequest) error
	AfterCreateOrganizationRole  func(ctx context.Context, user *limen.User, organization *Organization, role *OrganizationRole)

	BeforeUpdateOrganizationRole func(ctx context.Context, user *limen.User, organization *Organization, role *OrganizationRole, request *UpdateOrganizationRoleRequest) error
	AfterUpdateOrganizationRole  func(ctx context.Context, user *limen.User, organization *Organization, role *OrganizationRole)

	BeforeDeleteOrganizationRole func(ctx context.Context, user *limen.User, organization *Organization, role *OrganizationRole) error
	AfterDeleteOrganizationRole  func(ctx context.Context, user *limen.User, organization *Organization, role *OrganizationRole)
}

type ConfigOption func(*config)

// WithSlugGenerator derives the slug from the name and the client-provided slug,
// which is empty when none was sent.
func WithSlugGenerator(slugGenerator SlugGeneratorFunc) ConfigOption {
	return func(c *config) {
		c.slugGenerator = slugGenerator
	}
}

// WithSlugNormalization normalizes slugs before storage and lookup: lowercase,
// with runs of characters outside [a-z0-9] collapsed into single hyphens.
func WithSlugNormalization(enabled bool) ConfigOption {
	return func(c *config) {
		c.normalizeSlugs = enabled
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

func WithSendInvitationMail(sendInvitationMail func(ctx context.Context, data *SendInvitationMailData)) ConfigOption {
	return func(c *config) {
		c.sendInvitationMail = sendInvitationMail
	}
}

func WithCancelPendingInviteOnNewInvite(cancelPendingInviteOnNewInvite bool) ConfigOption {
	return func(c *config) {
		c.cancelPendingInviteOnNewInvite = cancelPendingInviteOnNewInvite
	}
}

func WithInvitationExpiration(invitationExpirationSeconds int) ConfigOption {
	return func(c *config) {
		c.invitationExpirationSeconds = invitationExpirationSeconds
	}
}

// WithMaxMembersPerOrganization sets the maximum number of members a organization can have.
// If set to 0, there is no limit.
func WithMaxMembersPerOrganization(maxMembersPerOrganization int) ConfigOption {
	return func(c *config) {
		c.maxMembersPerOrganization = maxMembersPerOrganization
	}
}

func WithMaxMembersPerOrganizationFunc(maxMembersPerOrganizationFunc MaxMembersPerOrganizationFunc) ConfigOption {
	return func(c *config) {
		c.maxMembersPerOrganization = maxMembersPerOrganizationFunc
	}
}

// WithCustomRoles enables organization-defined roles. When disabled the organization_roles
// table is not registered and the role management routes are not mounted.
func WithCustomRoles(enabled bool) ConfigOption {
	return func(c *config) {
		c.customRolesEnabled = enabled
	}
}

// WithMaxRolesPerOrganization sets the maximum number of custom roles an organization can define.
// If set to 0, there is no limit.
func WithMaxRolesPerOrganization(maxRolesPerOrganization int) ConfigOption {
	return func(c *config) {
		c.maxRolesPerOrganization = maxRolesPerOrganization
	}
}

// WithMaxRolesPerOrganizationFunc sets the limit per organization at request time.
// Returning 0 means no limit.
func WithMaxRolesPerOrganizationFunc(maxRolesPerOrganizationFunc MaxRolesPerOrganizationFunc) ConfigOption {
	return func(c *config) {
		c.maxRolesPerOrganization = maxRolesPerOrganizationFunc
	}
}
