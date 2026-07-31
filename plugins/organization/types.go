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

type config struct {
	accessControl                  *access.AccessControl
	roles                          []access.Role
	slugGenerator                  func(name string) string
	hooks                          Hooks
	ownerRole                      string
	maxOrgPerUser                  int
	maxMembersPerOrganization      any
	allowOrgCreation               func(ctx context.Context, user *limen.User) bool
	sendInvitationMail             func(ctx context.Context, data *SendInvitationMailData)
	cancelPendingInviteOnNewInvite bool
	invitationExpirationSeconds    int
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
