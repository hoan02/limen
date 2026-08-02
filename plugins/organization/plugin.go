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

	organizationSchema     *organizationSchema
	memberSchema           *memberSchema
	memberRoleSchema       *memberRoleSchema
	organizationRoleSchema *organizationRoleSchema
	invitationSchema       *invitationSchema
	hooks                  *Hooks
	responder              *limen.Responder
}

func New(opts ...ConfigOption) *organizationPlugin {
	config := &config{
		hooks:                          Hooks{},
		accessControl:                  access.New(defaultStatements),
		slugGenerator:                  defaultSlugGenerator,
		ownerRole:                      roleNameOwner,
		maxOrgPerUser:                  0,
		maxMembersPerOrganization:      0,
		invitationExpirationSeconds:    7 * 24 * 60 * 60,
		cancelPendingInviteOnNewInvite: false,
		customRolesEnabled:             false,
		maxRolesPerOrganization:        0,
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
	p.memberSchema = newMemberSchema(p)
	p.memberRoleSchema = newMemberRoleSchema(p.config.customRolesEnabled)
	p.organizationRoleSchema = newOrganizationRoleSchema()
	p.invitationSchema = newInvitationSchema(p)

	schemas := []limen.SchemaIntrospector{
		buildOrganizationTableDef(schema, p.organizationSchema),
		buildMemberTableDef(schema, p.memberSchema),
		buildMemberRoleTableDef(schema, p.memberRoleSchema, p.config.customRolesEnabled),
		buildInvitationTableDef(schema, p.invitationSchema),
		buildSessionActiveOrgExtension(schema),
	}

	if p.config.customRolesEnabled {
		schemas = append(schemas, buildOrganizationRoleTableDef(schema, p.organizationRoleSchema))
	}
	return schemas
}

func (p *organizationPlugin) Initialize(core *limen.LimenCore) error {
	p.core = core

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

	if err := p.validateRoles(); err != nil {
		return err
	}

	if p.getOwnerRole() == nil {
		return errors.New("organization: owner role must be provided")
	}

	return nil
}

func (p *organizationPlugin) validateRoles() error {
	seen := make(map[string]bool)
	for _, role := range p.config.roles {
		if seen[role.Name()] {
			return fmt.Errorf("role %q is defined multiple times", role.Name())
		}
		seen[role.Name()] = true
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

func (o *organizationPlugin) serializeEmbeddedUser(user *limen.User) map[string]any {
	if user == nil {
		return nil
	}

	if serializer, ok := o.config.embeddedUser.(EmbeddedUserSerializerFunc); ok {
		return serializer(user)
	}

	fields := embeddedFields(o.config.embeddedUser, defaultEmbeddedUserFields)
	return filterSerialized(o.core.SerializeModel(o.core.Schema.User, user), fields)
}

func (o *organizationPlugin) serializeEmbeddedOrganization(organization *Organization) map[string]any {
	if organization == nil {
		return nil
	}

	if serializer, ok := o.config.embeddedOrganization.(EmbeddedOrganizationSerializerFunc); ok {
		return serializer(organization)
	}

	fields := embeddedFields(o.config.embeddedOrganization, defaultEmbeddedOrganizationFields)
	return filterSerialized(o.core.SerializeModel(o.organizationSchema, organization), fields)
}

func embeddedFields(configured any, defaults []limen.SchemaField) []limen.SchemaField {
	if fields, ok := configured.([]limen.SchemaField); ok {
		return fields
	}
	return defaults
}

func filterSerialized(serialized map[string]any, fields []limen.SchemaField) map[string]any {
	filtered := make(map[string]any, len(fields))
	for _, field := range fields {
		if value, ok := serialized[string(field)]; ok {
			filtered[string(field)] = value
		}
	}
	return filtered
}
