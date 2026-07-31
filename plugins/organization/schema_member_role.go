package organization

import (
	"slices"
	"time"

	"github.com/thecodearcher/limen"
)

type MemberRole struct {
	ID                 any
	OrganizationID     any
	MemberID           any
	Role               *string
	OrganizationRoleID any

	CreatedAt time.Time

	raw map[string]any
}

func (r *MemberRole) Raw() map[string]any {
	return r.raw
}

const (
	MemberRoleSchemaTableName limen.SchemaTableName = "organization_member_roles"

	MemberRoleSchemaOrganizationIDField     limen.SchemaField = "organization_id"
	MemberRoleSchemaMemberIDField           limen.SchemaField = "member_id"
	MemberRoleSchemaRoleField               limen.SchemaField = "role"
	MemberRoleSchemaOrganizationRoleIDField limen.SchemaField = "organization_role_id"
)

type memberRoleSchema struct {
	limen.BaseSchema
	customRolesEnabled bool
}

func newMemberRoleSchema(customRolesEnabled bool) *memberRoleSchema {
	return &memberRoleSchema{customRolesEnabled: customRolesEnabled}
}

func (s *memberRoleSchema) GetOrganizationIDField() string {
	return s.GetField(MemberRoleSchemaOrganizationIDField)
}

func (s *memberRoleSchema) GetMemberIDField() string {
	return s.GetField(MemberRoleSchemaMemberIDField)
}
func (s *memberRoleSchema) GetRoleField() string {
	return s.GetField(MemberRoleSchemaRoleField)
}

func (s *memberRoleSchema) GetCreatedAtField() string {
	return s.GetField(limen.SchemaCreatedAtField)
}

func (s *memberRoleSchema) GetOrganizationRoleIDField() string {
	return s.GetField(MemberRoleSchemaOrganizationRoleIDField)
}

func (s *memberRoleSchema) ToStorage(data limen.Model) map[string]any {
	memberRole := data.(*MemberRole)
	payload := map[string]any{
		s.GetOrganizationIDField(): memberRole.OrganizationID,
		s.GetMemberIDField():       memberRole.MemberID,
		s.GetRoleField():           memberRole.Role,
	}

	if s.customRolesEnabled {
		payload[s.GetOrganizationRoleIDField()] = memberRole.OrganizationRoleID
	}
	return payload
}

func (s *memberRoleSchema) FromStorage(data map[string]any) limen.Model {
	memberRole := &MemberRole{
		ID:             data[s.GetIDField()],
		OrganizationID: limen.GetValue[any](data[s.GetOrganizationIDField()]),
		MemberID:       limen.GetValue[any](data[s.GetMemberIDField()]),
		Role:           limen.GetNullableValue[string](data[s.GetRoleField()]),
		CreatedAt:      limen.GetValue[time.Time](data[s.GetCreatedAtField()]),
		raw:            data,
	}

	if s.customRolesEnabled {
		memberRole.OrganizationRoleID = limen.GetValue[any](data[s.GetOrganizationRoleIDField()])
	}
	return memberRole
}

func buildMemberRoleTableDef(schemaConfig *limen.SchemaConfig, schema *memberRoleSchema, customRolesEnabled bool) *limen.SchemaDefinition {
	opts := []limen.SchemaDefinitionOption{
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(MemberRoleSchemaMemberIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(MemberRoleSchemaOrganizationIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(MemberRoleSchemaRoleField, limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaCreatedAtField(),

		limen.WithSchemaUniqueIndex("idx_organization_member_roles_member_role", []limen.SchemaField{
			MemberRoleSchemaMemberIDField,
			MemberRoleSchemaRoleField,
		}),

		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_organization_member_roles_member",
			Column:           MemberRoleSchemaMemberIDField,
			ReferencedSchema: limen.SchemaName(MemberSchemaTableName),
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),

		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_organization_member_roles_organization",
			Column:           MemberRoleSchemaOrganizationIDField,
			ReferencedSchema: limen.SchemaName(OrganizationSchemaTableName),
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
		limen.WithDisablePublicID(),
	}

	if customRolesEnabled {
		opts = slices.Insert(opts, 3,
			limen.WithSchemaField(MemberRoleSchemaOrganizationRoleIDField, schemaConfig.GetIDColumnType(), limen.WithNullable(true)),
		)
		opts = append(opts,
			limen.WithSchemaUniqueIndex("idx_organization_member_roles_member_dynamic_role", []limen.SchemaField{
				MemberRoleSchemaMemberIDField,
				MemberRoleSchemaOrganizationRoleIDField,
			}),
			limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
				Name:             "fk_organization_member_roles_organization_role",
				Column:           MemberRoleSchemaOrganizationRoleIDField,
				ReferencedSchema: limen.SchemaName(OrganizationRoleSchemaTableName),
				ReferencedField:  limen.SchemaIDField,
				OnDelete:         limen.FKActionCascade,
				OnUpdate:         limen.FKActionCascade,
			}),
		)
	}

	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(MemberRoleSchemaTableName),
		MemberRoleSchemaTableName,
		schema,
		opts...,
	)
}
