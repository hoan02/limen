package organization

import (
	"encoding/json"
	"time"

	"github.com/thecodearcher/limen"
)

type Invitation struct {
	ID             any
	OrganizationID any
	Email          string
	Roles          []string
	Status         InvitationStatus
	TokenHash      string
	InviterID      any
	ExpiresAt      time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	raw map[string]any
}

func (i *Invitation) Raw() map[string]any {
	return i.raw
}

const (
	InvitationSchemaTableName limen.SchemaTableName = "organization_invitations"

	InvitationSchemaOrganizationIDField limen.SchemaField = "organization_id"
	InvitationSchemaEmailField          limen.SchemaField = "email"
	InvitationSchemaRolesField          limen.SchemaField = "roles"
	InvitationSchemaStatusField         limen.SchemaField = "status"
	InvitationSchemaTokenHashField      limen.SchemaField = "token_hash"
	InvitationSchemaInviterIDField      limen.SchemaField = "inviter_id"
	InvitationSchemaExpiresAtField      limen.SchemaField = "expires_at"
)

type invitationSchema struct {
	limen.BaseSchema
}

func newInvitationSchema() *invitationSchema {
	return &invitationSchema{BaseSchema: limen.BaseSchema{}}
}

func (s *invitationSchema) GetOrganizationIDField() string {
	return s.GetField(InvitationSchemaOrganizationIDField)
}
func (s *invitationSchema) GetEmailField() string { return s.GetField(InvitationSchemaEmailField) }
func (s *invitationSchema) GetRolesField() string { return s.GetField(InvitationSchemaRolesField) }
func (s *invitationSchema) GetStatusField() string {
	return s.GetField(InvitationSchemaStatusField)
}
func (s *invitationSchema) GetTokenHashField() string {
	return s.GetField(InvitationSchemaTokenHashField)
}
func (s *invitationSchema) GetInviterIDField() string {
	return s.GetField(InvitationSchemaInviterIDField)
}
func (s *invitationSchema) GetExpiresAtField() string {
	return s.GetField(InvitationSchemaExpiresAtField)
}
func (s *invitationSchema) GetCreatedAtField() string {
	return s.GetField(limen.SchemaCreatedAtField)
}
func (s *invitationSchema) GetUpdatedAtField() string {
	return s.GetField(limen.SchemaUpdatedAtField)
}

func (s *invitationSchema) ToStorage(data limen.Model) map[string]any {
	inv := data.(*Invitation)
	payload := map[string]any{
		s.GetOrganizationIDField(): inv.OrganizationID,
		s.GetEmailField():          inv.Email,
		s.GetStatusField():         string(inv.Status),
		s.GetTokenHashField():      inv.TokenHash,
		s.GetInviterIDField():      inv.InviterID,
		s.GetExpiresAtField():      inv.ExpiresAt,
	}

	if inv.Roles != nil {
		if b, err := json.Marshal(inv.Roles); err == nil {
			payload[s.GetRolesField()] = string(b)
		}
	}
	return payload
}

func (s *invitationSchema) FromStorage(data map[string]any) limen.Model {
	return &Invitation{
		ID:             data[s.GetIDField()],
		OrganizationID: limen.GetValue[any](data[s.GetOrganizationIDField()]),
		Email:          limen.GetValue[string](data[s.GetEmailField()]),
		Roles:          limen.ParseJSONFromStorage[[]string](data, s.GetRolesField()),
		Status:         InvitationStatus(limen.GetValue[string](data[s.GetStatusField()])),
		TokenHash:      limen.GetValue[string](data[s.GetTokenHashField()]),
		InviterID:      limen.GetValue[any](data[s.GetInviterIDField()]),
		ExpiresAt:      limen.GetValue[time.Time](data[s.GetExpiresAtField()]),
		CreatedAt:      limen.GetValue[time.Time](data[s.GetCreatedAtField()]),
		UpdatedAt:      limen.GetValue[time.Time](data[s.GetUpdatedAtField()]),
		raw:            data,
	}
}

func buildInvitationTableDef(schemaConfig *limen.SchemaConfig, schema *invitationSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(InvitationSchemaTableName),
		InvitationSchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(InvitationSchemaOrganizationIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(InvitationSchemaEmailField, limen.ColumnTypeString),
		limen.WithSchemaField(InvitationSchemaRolesField, limen.ColumnTypeText, limen.WithNullable(true)),
		limen.WithSchemaField(InvitationSchemaStatusField, limen.ColumnTypeString),
		limen.WithSchemaField(InvitationSchemaTokenHashField, limen.ColumnTypeText),
		limen.WithSchemaField(InvitationSchemaInviterIDField, schemaConfig.GetIDColumnType(), limen.WithNullable(true)),
		limen.WithSchemaField(InvitationSchemaExpiresAtField, limen.ColumnTypeTime),
		limen.WithSchemaCreatedAtField(),
		limen.WithSchemaUpdatedAtField(),

		limen.WithSchemaUniqueIndex("idx_organization_invitations_token_hash", []limen.SchemaField{
			InvitationSchemaTokenHashField,
		}),
		limen.WithSchemaIndex("idx_organization_invitations_org", []limen.SchemaField{
			InvitationSchemaOrganizationIDField,
		}),
		limen.WithSchemaIndex("idx_organization_invitations_email", []limen.SchemaField{
			InvitationSchemaEmailField,
		}),

		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_organization_invitations_organization",
			Column:           InvitationSchemaOrganizationIDField,
			ReferencedSchema: limen.SchemaName(OrganizationSchemaTableName),
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_organization_invitations_inviter",
			Column:           InvitationSchemaInviterIDField,
			ReferencedSchema: limen.CoreSchemaUsers,
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionSetNull,
			OnUpdate:         limen.FKActionCascade,
		}),
	)
}
