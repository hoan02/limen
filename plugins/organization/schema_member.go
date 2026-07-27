package organization

import (
	"time"

	"github.com/thecodearcher/limen"
)

type Member struct {
	ID             any
	OrganizationID any
	UserID         any

	CreatedAt time.Time
	UpdatedAt time.Time

	raw map[string]any
}

func (m *Member) Raw() map[string]any {
	return m.raw
}

const (
	MemberSchemaTableName limen.SchemaTableName = "organization_members"

	MemberSchemaOrganizationIDField limen.SchemaField = "organization_id"
	MemberSchemaUserIDField         limen.SchemaField = "user_id"
)

type memberSchema struct {
	limen.BaseSchema
}

func newMemberSchema() *memberSchema {
	return &memberSchema{BaseSchema: limen.BaseSchema{}}
}

func (s *memberSchema) GetOrganizationIDField() string {
	return s.GetField(MemberSchemaOrganizationIDField)
}
func (s *memberSchema) GetUserIDField() string    { return s.GetField(MemberSchemaUserIDField) }
func (s *memberSchema) GetCreatedAtField() string { return s.GetField(limen.SchemaCreatedAtField) }
func (s *memberSchema) GetUpdatedAtField() string { return s.GetField(limen.SchemaUpdatedAtField) }

func (s *memberSchema) ToStorage(data limen.Model) map[string]any {
	member := data.(*Member)
	return map[string]any{
		s.GetOrganizationIDField(): member.OrganizationID,
		s.GetUserIDField():         member.UserID,
	}
}

func (s *memberSchema) FromStorage(data map[string]any) limen.Model {
	return &Member{
		ID:             data[s.GetIDField()],
		OrganizationID: limen.GetValue[any](data[s.GetOrganizationIDField()]),
		UserID:         limen.GetValue[any](data[s.GetUserIDField()]),
		CreatedAt:      limen.GetValue[time.Time](data[s.GetCreatedAtField()]),
		UpdatedAt:      limen.GetValue[time.Time](data[s.GetUpdatedAtField()]),
		raw:            data,
	}
}

func buildMemberTableDef(schemaConfig *limen.SchemaConfig, schema *memberSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(MemberSchemaTableName),
		MemberSchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(MemberSchemaOrganizationIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(MemberSchemaUserIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaCreatedAtField(),
		limen.WithSchemaUpdatedAtField(),

		limen.WithSchemaUniqueIndex("idx_organization_members_org_user", []limen.SchemaField{
			MemberSchemaOrganizationIDField,
			MemberSchemaUserIDField,
		}),

		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_organization_members_organization",
			Column:           MemberSchemaOrganizationIDField,
			ReferencedSchema: limen.SchemaName(OrganizationSchemaTableName),
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_organization_members_user",
			Column:           MemberSchemaUserIDField,
			ReferencedSchema: limen.CoreSchemaUsers,
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
	)
}
