package organization

import (
	"encoding/json"
	"maps"
	"time"

	"github.com/thecodearcher/limen"
)

type OrganizationRole struct {
	ID             any
	OrganizationID any
	Name           string
	Permissions    map[string][]string
	Description    *string

	CreatedAt time.Time
	UpdatedAt time.Time

	raw map[string]any
}

func (r *OrganizationRole) Raw() map[string]any {
	return r.raw
}

const (
	OrganizationRoleSchemaTableName limen.SchemaTableName = "organization_roles"

	OrganizationRoleSchemaOrganizationIDField limen.SchemaField = "organization_id"
	OrganizationRoleSchemaNameField           limen.SchemaField = "name"
	OrganizationRoleSchemaPermissionsField    limen.SchemaField = "permissions"
	OrganizationRoleSchemaDescriptionField    limen.SchemaField = "description"
)

type organizationRoleSchema struct {
	limen.BaseSchema
}

func newOrganizationRoleSchema() *organizationRoleSchema {
	return &organizationRoleSchema{BaseSchema: limen.BaseSchema{Serializer: organizationRoleSerializer()}}
}

func organizationRoleSerializer() limen.ModelTransformer {
	return limen.ModelTransformer(func(data limen.Model) map[string]any {
		role := data.(*OrganizationRole)
		payload := maps.Clone(role.raw)
		if payload == nil {
			payload = make(map[string]any)
		}

		permissions := role.Permissions
		if permissions == nil {
			permissions = make(map[string][]string)
		}
		payload["permissions"] = permissions

		delete(payload, "organization_id")
		return payload
	})
}

func (s *organizationRoleSchema) GetOrganizationIDField() string {
	return s.GetField(OrganizationRoleSchemaOrganizationIDField)
}
func (s *organizationRoleSchema) GetNameField() string {
	return s.GetField(OrganizationRoleSchemaNameField)
}
func (s *organizationRoleSchema) GetPermissionsField() string {
	return s.GetField(OrganizationRoleSchemaPermissionsField)
}
func (s *organizationRoleSchema) GetDescriptionField() string {
	return s.GetField(OrganizationRoleSchemaDescriptionField)
}
func (s *organizationRoleSchema) GetCreatedAtField() string {
	return s.GetField(limen.SchemaCreatedAtField)
}
func (s *organizationRoleSchema) GetUpdatedAtField() string {
	return s.GetField(limen.SchemaUpdatedAtField)
}

func (s *organizationRoleSchema) ToStorage(data limen.Model) map[string]any {
	role := data.(*OrganizationRole)
	payload := map[string]any{
		s.GetOrganizationIDField(): role.OrganizationID,
		s.GetNameField():           role.Name,
		s.GetDescriptionField():    role.Description,
	}

	if role.Permissions != nil {
		if b, err := json.Marshal(role.Permissions); err == nil {
			payload[s.GetPermissionsField()] = string(b)
		}
	}
	return payload
}

func (s *organizationRoleSchema) FromStorage(data map[string]any) limen.Model {
	return &OrganizationRole{
		ID:             data[s.GetIDField()],
		OrganizationID: limen.GetValue[any](data[s.GetOrganizationIDField()]),
		Name:           limen.GetValue[string](data[s.GetNameField()]),
		Permissions:    limen.ParseJSONFromStorage[map[string][]string](data, s.GetPermissionsField()),
		Description:    limen.GetNullableValue[string](data[s.GetDescriptionField()]),
		CreatedAt:      limen.GetValue[time.Time](data[s.GetCreatedAtField()]),
		UpdatedAt:      limen.GetValue[time.Time](data[s.GetUpdatedAtField()]),
		raw:            data,
	}
}

func buildOrganizationRoleTableDef(schemaConfig *limen.SchemaConfig, schema *organizationRoleSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(OrganizationRoleSchemaTableName),
		OrganizationRoleSchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(OrganizationRoleSchemaOrganizationIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(OrganizationRoleSchemaNameField, limen.ColumnTypeString),
		limen.WithSchemaField(OrganizationRoleSchemaPermissionsField, limen.ColumnTypeMapStringAny, limen.WithNullable(true)),
		limen.WithSchemaField(OrganizationRoleSchemaDescriptionField, limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaCreatedAtField(),
		limen.WithSchemaUpdatedAtField(),

		limen.WithSchemaUniqueIndex("idx_organization_roles_org_name", []limen.SchemaField{
			OrganizationRoleSchemaOrganizationIDField,
			OrganizationRoleSchemaNameField,
		}),

		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_organization_roles_organization",
			Column:           OrganizationRoleSchemaOrganizationIDField,
			ReferencedSchema: limen.SchemaName(OrganizationSchemaTableName),
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
	)
}
