package organization

import (
	"github.com/thecodearcher/limen"
)

const SessionSchemaActiveOrganizationIDField limen.SchemaField = "active_organization_id"

type sessionSchema struct {
	*limen.SessionSchema
}

func (s *sessionSchema) GetActiveOrganizationIDField() string {
	return s.GetField(SessionSchemaActiveOrganizationIDField)
}

func buildSessionActiveOrgExtension(schemaConfig *limen.SchemaConfig) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForExtension(
		limen.CoreSchemaSessions,
		&sessionSchema{SessionSchema: schemaConfig.Session},
		limen.WithSchemaField(SessionSchemaActiveOrganizationIDField, schemaConfig.GetIDColumnType(), limen.WithNullable(true)),
		limen.WithSchemaIndex("idx_sessions_active_organization", []limen.SchemaField{SessionSchemaActiveOrganizationIDField}),
	)
}
