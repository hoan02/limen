package organization

import (
	"encoding/json"
	"time"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type Invitation struct {
	ID             any
	OrganizationID any
	Email          string
	Roles          []any
	Status         InvitationStatus
	Token          string
	InviterID      any
	ExpiresAt      *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	Organization  *Organization
	Inviter       *limen.User
	ResolvedRoles []*access.Role

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
	InvitationSchemaTokenField          limen.SchemaField = "token"
	InvitationSchemaInviterIDField      limen.SchemaField = "inviter_id"
	InvitationSchemaExpiresAtField      limen.SchemaField = "expires_at"
)

type invitationSchema struct {
	limen.BaseSchema
}

func newInvitationSchema(plugin *organizationPlugin) *invitationSchema {
	return &invitationSchema{BaseSchema: limen.BaseSchema{Serializer: invitationSerializer(plugin)}}
}

func invitationSerializer(plugin *organizationPlugin) limen.ModelTransformer {
	return func(data limen.Model) map[string]any {
		inv := data.(*Invitation)
		payload := map[string]any{
			"id":         inv.ID,
			"email":      inv.Email,
			"status":     inv.Status,
			"expires_at": inv.ExpiresAt,
			"is_expired": inv.ExpiresAt != nil && inv.ExpiresAt.Before(time.Now()),
			"created_at": inv.CreatedAt,
			"updated_at": inv.UpdatedAt,
		}

		if inv.ResolvedRoles != nil {
			payload["roles"] = SortedRoleNames(inv.ResolvedRoles)
		}

		if inv.Organization != nil {
			payload["organization"] = plugin.serializeEmbeddedOrganization(inv.Organization)
		}

		if inv.Inviter != nil {
			payload["inviter"] = plugin.serializeEmbeddedUser(inv.Inviter)
		}

		return payload
	}
}

func (s *invitationSchema) GetOrganizationIDField() string {
	return s.GetField(InvitationSchemaOrganizationIDField)
}
func (s *invitationSchema) GetEmailField() string {
	return s.GetField(InvitationSchemaEmailField)
}
func (s *invitationSchema) GetRolesField() string {
	return s.GetField(InvitationSchemaRolesField)
}
func (s *invitationSchema) GetStatusField() string {
	return s.GetField(InvitationSchemaStatusField)
}
func (s *invitationSchema) GetTokenField() string {
	return s.GetField(InvitationSchemaTokenField)
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
		s.GetTokenField():          inv.Token,
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
		Roles:          limen.ParseJSONFromStorage[[]any](data, s.GetRolesField()),
		Status:         InvitationStatus(limen.GetValue[string](data[s.GetStatusField()])),
		Token:          limen.GetValue[string](data[s.GetTokenField()]),
		InviterID:      limen.GetValue[any](data[s.GetInviterIDField()]),
		ExpiresAt:      limen.GetNullableValue[time.Time](data[s.GetExpiresAtField()]),
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
		limen.WithSchemaField(InvitationSchemaInviterIDField, schemaConfig.GetIDColumnType(), limen.WithNullable(true)),
		limen.WithSchemaField(InvitationSchemaEmailField, limen.ColumnTypeString),
		limen.WithSchemaField(InvitationSchemaRolesField, limen.ColumnTypeText, limen.WithNullable(true)),
		limen.WithSchemaField(InvitationSchemaStatusField, limen.ColumnTypeString),
		limen.WithSchemaField(InvitationSchemaTokenField, limen.ColumnTypeText),
		limen.WithSchemaField(InvitationSchemaExpiresAtField, limen.ColumnTypeTime, limen.WithNullable(true)),
		limen.WithSchemaCreatedAtField(),
		limen.WithSchemaUpdatedAtField(),

		limen.WithSchemaUniqueIndex("idx_organization_invitations_token", []limen.SchemaField{
			InvitationSchemaTokenField,
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
