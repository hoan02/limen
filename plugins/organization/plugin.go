package organization

import (
	"errors"

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
	hooks                  *Hooks
}

func New(opts ...ConfigOption) *organizationPlugin {
	config := &config{
		hooks:         Hooks{},
		accessControl: access.New(defaultStatements),
		ownerRole:     roleNameOwner,
		maxOrgPerUser: 0,
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
	p.memberSchema = newMemberSchema()
	p.memberRoleSchema = newMemberRoleSchema()
	p.organizationRoleSchema = newOrganizationRoleSchema()
	p.invitationSchema = newInvitationSchema()

	return []limen.SchemaIntrospector{
		buildOrganizationTableDef(schema, p.organizationSchema),
		buildOrganizationRoleTableDef(schema, p.organizationRoleSchema),
		buildMemberTableDef(schema, p.memberSchema),
		buildMemberRoleTableDef(schema, p.memberRoleSchema),
		buildInvitationTableDef(schema, p.invitationSchema),
		buildSessionActiveOrgExtension(schema),
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
