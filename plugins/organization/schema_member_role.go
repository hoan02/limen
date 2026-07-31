package organization

import (
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
}

func newMemberRoleSchema() *memberRoleSchema {
	return &memberRoleSchema{BaseSchema: limen.BaseSchema{}}
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
	return map[string]any{
		s.GetOrganizationIDField():     memberRole.OrganizationID,
		s.GetMemberIDField():           memberRole.MemberID,
		s.GetRoleField():               memberRole.Role,
		s.GetOrganizationRoleIDField(): memberRole.OrganizationRoleID,
	}
}

func (s *memberRoleSchema) FromStorage(data map[string]any) limen.Model {
	return &MemberRole{
		ID:                 data[s.GetIDField()],
		OrganizationID:     limen.GetValue[any](data[s.GetOrganizationIDField()]),
		MemberID:           limen.GetValue[any](data[s.GetMemberIDField()]),
		Role:               limen.GetNullableValue[string](data[s.GetRoleField()]),
		OrganizationRoleID: limen.GetValue[any](data[s.GetOrganizationRoleIDField()]),
		CreatedAt:          limen.GetValue[time.Time](data[s.GetCreatedAtField()]),
		raw:                data,
	}
}

func buildMemberRoleTableDef(schemaConfig *limen.SchemaConfig, schema *memberRoleSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(MemberRoleSchemaTableName),
		MemberRoleSchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(MemberRoleSchemaMemberIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(MemberRoleSchemaOrganizationIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(MemberRoleSchemaOrganizationRoleIDField, schemaConfig.GetIDColumnType(), limen.WithNullable(true)),
		limen.WithSchemaField(MemberRoleSchemaRoleField, limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaCreatedAtField(),

		limen.WithSchemaUniqueIndex("idx_organization_member_roles_member_role", []limen.SchemaField{
			MemberRoleSchemaMemberIDField,
			MemberRoleSchemaRoleField,
		}),

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
	)
}
