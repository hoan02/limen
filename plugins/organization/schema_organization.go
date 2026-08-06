package organization

import (
	"encoding/json"
	"time"

	"github.com/thecodearcher/limen"
)

type Organization struct {
	ID       any
	Name     string
	Slug     string
	Logo     *string
	Metadata map[string]any

	CreatedAt time.Time
	UpdatedAt time.Time

	raw map[string]any
}

func (o *Organization) Raw() map[string]any {
	return o.raw
}

const (
	OrganizationSchemaTableName limen.SchemaTableName = "organizations"
	OrganizationSchemaName      limen.SchemaName      = limen.SchemaName(OrganizationSchemaTableName)

	OrganizationSchemaNameField     limen.SchemaField = "name"
	OrganizationSchemaUserIDField   limen.SchemaField = "user_id"
	OrganizationSchemaSlugField     limen.SchemaField = "slug"
	OrganizationSchemaLogoField     limen.SchemaField = "logo"
	OrganizationSchemaMetadataField limen.SchemaField = "metadata"
)

type organizationSchema struct {
	limen.BaseSchema
}

func newOrganizationSchema() *organizationSchema {
	return &organizationSchema{}
}

func (s *organizationSchema) GetNameField() string { return s.GetField(OrganizationSchemaNameField) }
func (s *organizationSchema) GetSlugField() string { return s.GetField(OrganizationSchemaSlugField) }
func (s *organizationSchema) GetLogoField() string { return s.GetField(OrganizationSchemaLogoField) }
func (s *organizationSchema) GetMetadataField() string {
	return s.GetField(OrganizationSchemaMetadataField)
}
func (s *organizationSchema) GetCreatedAtField() string {
	return s.GetField(limen.SchemaCreatedAtField)
}
func (s *organizationSchema) GetUpdatedAtField() string {
	return s.GetField(limen.SchemaUpdatedAtField)
}

func (s *organizationSchema) ToStorage(data limen.Model) map[string]any {
	org := data.(*Organization)
	payload := map[string]any{
		s.GetNameField(): org.Name,
		s.GetSlugField(): org.Slug,
		s.GetLogoField(): org.Logo,
	}

	if org.Metadata != nil {
		if b, err := json.Marshal(org.Metadata); err == nil {
			payload[s.GetMetadataField()] = string(b)
		}
	}
	return payload
}

func (s *organizationSchema) FromStorage(data map[string]any) limen.Model {
	return &Organization{
		ID:        data[s.GetIDField()],
		Name:      limen.GetValue[string](data[s.GetNameField()]),
		Slug:      limen.GetValue[string](data[s.GetSlugField()]),
		Logo:      limen.GetNullableValue[string](data[s.GetLogoField()]),
		Metadata:  limen.ParseJSONFromStorage[map[string]any](data, s.GetMetadataField()),
		CreatedAt: limen.GetValue[time.Time](data[s.GetCreatedAtField()]),
		UpdatedAt: limen.GetValue[time.Time](data[s.GetUpdatedAtField()]),
		raw:       data,
	}
}

func buildOrganizationTableDef(schemaConfig *limen.SchemaConfig, schema *organizationSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		OrganizationSchemaName,
		OrganizationSchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(OrganizationSchemaNameField, limen.ColumnTypeString),
		limen.WithSchemaField(OrganizationSchemaSlugField, limen.ColumnTypeString),
		limen.WithSchemaField(OrganizationSchemaLogoField, limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaField(OrganizationSchemaMetadataField, limen.ColumnTypeMapStringAny, limen.WithNullable(true)),
		limen.WithSchemaCreatedAtField(),
		limen.WithSchemaUpdatedAtField(),

		limen.WithSchemaUniqueIndex("idx_organizations_slug", []limen.SchemaField{OrganizationSchemaSlugField}),
	)
}
