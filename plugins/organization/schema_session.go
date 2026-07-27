package organization

import (
	"github.com/thecodearcher/limen"
)

const (
	SessionSchemaActiveOrganizationIDField limen.SchemaField = "active_organization_id"
)

type sessionSchema struct {
	*limen.SessionSchema
}

func (s *sessionSchema) GetActiveOrganizationIDField() string {
	return s.GetField(SessionSchemaActiveOrganizationIDField)
}

func buildSessionActiveOrgExtension(schemaConfig *limen.SchemaConfig) *limen.SchemaDefinition {
	extended := &sessionSchema{SessionSchema: schemaConfig.Session}
	return limen.NewSchemaDefinitionForExtension(
		limen.CoreSchemaSessions,
		extended,
		limen.WithSchemaField(SessionSchemaActiveOrganizationIDField, schemaConfig.GetIDColumnType(), limen.WithNullable(true)),
	)
}
