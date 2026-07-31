package organization

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type organizationPlugin struct {
	config *config

	core *limen.LimenCore
	db   *limen.DatabaseActionHelper

	organizationSchema     *organizationSchema
	memberSchema           *memberSchema
	memberRoleSchema       *memberRoleSchema
	organizationRoleSchema *organizationRoleSchema
	invitationSchema       *invitationSchema
	sessionSchema          *limen.SessionSchema
	hooks                  *Hooks
	responder              *limen.Responder
}

func New(opts ...ConfigOption) *organizationPlugin {
	config := &config{
		hooks:                          Hooks{},
		accessControl:                  access.New(defaultStatements),
		ownerRole:                      roleNameOwner,
		maxOrgPerUser:                  0,
		maxMembersPerOrganization:      0,
		invitationExpirationSeconds:    7 * 24 * 60 * 60,
		cancelPendingInviteOnNewInvite: false,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &organizationPlugin{config: config, hooks: &config.hooks}
}

func (p *organizationPlugin) Name() limen.PluginName {
	return limen.PluginOrganization
}

func (p *organizationPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	p.organizationSchema = newOrganizationSchema()
	p.memberSchema = newMemberSchema(schema, p)
	p.memberRoleSchema = newMemberRoleSchema()
	p.organizationRoleSchema = newOrganizationRoleSchema()
	p.invitationSchema = newInvitationSchema(p)
	p.sessionSchema = schema.Session
	return []limen.SchemaIntrospector{
		buildOrganizationTableDef(schema, p.organizationSchema),
		buildOrganizationRoleTableDef(schema, p.organizationRoleSchema),
		buildMemberTableDef(schema, p.memberSchema),
		buildMemberRoleTableDef(schema, p.memberRoleSchema),
		buildInvitationTableDef(schema, p.invitationSchema),
	}
}

func (p *organizationPlugin) Initialize(core *limen.LimenCore) error {
	p.core = core
	p.db = core.DBAction

	if p.config.accessControl == nil {
		return errors.New("organization: access control must be provided")
	}

	if len(p.config.roles) == 0 {
		roles, err := DefaultRoles(p.config.accessControl)
		if err != nil {
			return err
		}
		p.config.roles = roles
	}

	if p.getOwnerRole() == nil {
		return errors.New("organization: owner role must be provided")
	}

	return nil
}

func (p *organizationPlugin) getOwnerRole() *access.Role {
	for _, role := range p.config.roles {
		if role.Name() == p.config.ownerRole {
			return &role
		}
	}
	return nil
}

func (p *organizationPlugin) isOwnerRole(role *access.Role) bool {
	return strings.EqualFold(role.Name(), p.config.ownerRole)
}

func (o *organizationPlugin) clientOrganizationID(org *Organization) string {
	if encoded, ok := o.core.EncodePublicID(o.organizationSchema, org); ok {
		return encoded
	}
	return fmt.Sprintf("%v", org.ID)
}

func perms(specs ...string) access.Permissions {
	return access.P(specs...)
}
