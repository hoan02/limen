package organization

import (
	"time"

	"github.com/thecodearcher/limen"
)

type MemberRole struct {
	ID             any
	OrganizationID any
	MemberID       any
	Role           string

	CreatedAt time.Time

	raw map[string]any
}

func (r *MemberRole) Raw() map[string]any {
	return r.raw
}

const (
	MemberRoleSchemaTableName limen.SchemaTableName = "organization_member_roles"

	MemberRoleSchemaMemberIDField limen.SchemaField = "member_id"
	MemberRoleSchemaRoleField     limen.SchemaField = "role"
)

type memberRoleSchema struct {
	limen.BaseSchema
}

func newMemberRoleSchema() *memberRoleSchema {
	return &memberRoleSchema{BaseSchema: limen.BaseSchema{}}
}

func (s *memberRoleSchema) GetMemberIDField() string {
	return s.GetField(MemberRoleSchemaMemberIDField)
}
func (s *memberRoleSchema) GetRoleField() string { return s.GetField(MemberRoleSchemaRoleField) }
func (s *memberRoleSchema) GetCreatedAtField() string {
	return s.GetField(limen.SchemaCreatedAtField)
}

func (s *memberRoleSchema) ToStorage(data limen.Model) map[string]any {
	memberRole := data.(*MemberRole)
	return map[string]any{
		s.GetMemberIDField(): memberRole.MemberID,
		s.GetRoleField():     memberRole.Role,
	}
}

func (s *memberRoleSchema) FromStorage(data map[string]any) limen.Model {
	return &MemberRole{
		ID:        data[s.GetIDField()],
		MemberID:  limen.GetValue[any](data[s.GetMemberIDField()]),
		Role:      limen.GetValue[string](data[s.GetRoleField()]),
		CreatedAt: limen.GetValue[time.Time](data[s.GetCreatedAtField()]),
		raw:       data,
	}
}

func buildMemberRoleTableDef(schemaConfig *limen.SchemaConfig, schema *memberRoleSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(MemberRoleSchemaTableName),
		MemberRoleSchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(MemberRoleSchemaMemberIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(MemberRoleSchemaRoleField, limen.ColumnTypeString),
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
		limen.WithDisablePublicID(),
	)
}
