package apikey

import (
	"encoding/json"
	"time"

	"github.com/thecodearcher/limen"
)

type ApiKey struct {
	ID      any
	Name    string
	Profile string

	// Principal — who the key acts as; PrincipalID is resolved via PrincipalType.
	CreatedByUserID any
	PrincipalType   PrincipalType
	PrincipalID     any

	KeyHash string
	Prefix  *string
	Last4   string

	Permissions map[string][]string

	Enabled   bool
	ExpiresAt *time.Time

	RateLimitMax          *int32
	RateLimitWindowMS     *int64
	RateLimitRequestCount *int32

	LastUsedAt *time.Time
	Metadata   map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time

	raw map[string]any
}

func (a *ApiKey) Raw() map[string]any {
	return a.raw
}

func (a *ApiKey) RateLimitEnabled() bool {
	return a.RateLimitMax != nil && a.RateLimitWindowMS != nil
}

const (
	APIKeySchemaTableName limen.SchemaTableName = "api_keys"

	APIKeySchemaNameField                  limen.SchemaField = "name"
	APIKeySchemaProfileField               limen.SchemaField = "profile"
	APIKeySchemaCreatedByField             limen.SchemaField = "created_by_user_id"
	APIKeySchemaPrincipalTypeField         limen.SchemaField = "principal_type" // user, member, organization, ...
	APIKeySchemaPrincipalIDField           limen.SchemaField = "principal_id"
	APIKeySchemaKeyHashField               limen.SchemaField = "key_hash"
	APIKeySchemaPrefixField                limen.SchemaField = "prefix"
	APIKeySchemaLast4Field                 limen.SchemaField = "last4"
	APIKeySchemaPermissionsField           limen.SchemaField = "permissions"
	APIKeySchemaEnabledField               limen.SchemaField = "enabled"
	APIKeySchemaExpiresAtField             limen.SchemaField = "expires_at"
	APIKeySchemaRateLimitMaxField          limen.SchemaField = "rate_limit_max"
	APIKeySchemaRateLimitWindowField       limen.SchemaField = "rate_limit_window_ms"
	APIKeySchemaRateLimitRequestCountField limen.SchemaField = "rate_limit_request_count"
	APIKeySchemaLastUsedAtField            limen.SchemaField = "last_used_at"
	APIKeySchemaMetadataField              limen.SchemaField = "metadata"
)

type apiKeySchema struct {
	limen.BaseSchema
}

func newAPIKeySchema() *apiKeySchema {
	return &apiKeySchema{BaseSchema: limen.BaseSchema{}}
}

func (s *apiKeySchema) GetNameField() string {
	return s.GetField(APIKeySchemaNameField)
}

func (s *apiKeySchema) GetProfileField() string {
	return s.GetField(APIKeySchemaProfileField)
}

func (s *apiKeySchema) GetCreatedByField() string {
	return s.GetField(APIKeySchemaCreatedByField)
}

func (s *apiKeySchema) GetPrincipalTypeField() string {
	return s.GetField(APIKeySchemaPrincipalTypeField)
}

func (s *apiKeySchema) GetPrincipalIDField() string {
	return s.GetField(APIKeySchemaPrincipalIDField)
}

func (s *apiKeySchema) GetKeyHashField() string {
	return s.GetField(APIKeySchemaKeyHashField)
}

func (s *apiKeySchema) GetPrefixField() string {
	return s.GetField(APIKeySchemaPrefixField)
}

func (s *apiKeySchema) GetLast4Field() string {
	return s.GetField(APIKeySchemaLast4Field)
}

func (s *apiKeySchema) GetPermissionsField() string {
	return s.GetField(APIKeySchemaPermissionsField)
}

func (s *apiKeySchema) GetEnabledField() string {
	return s.GetField(APIKeySchemaEnabledField)
}

func (s *apiKeySchema) GetExpiresAtField() string {
	return s.GetField(APIKeySchemaExpiresAtField)
}

func (s *apiKeySchema) GetRateLimitMaxField() string {
	return s.GetField(APIKeySchemaRateLimitMaxField)
}

func (s *apiKeySchema) GetRateLimitWindowField() string {
	return s.GetField(APIKeySchemaRateLimitWindowField)
}

func (s *apiKeySchema) GetRateLimitRequestCountField() string {
	return s.GetField(APIKeySchemaRateLimitRequestCountField)
}

func (s *apiKeySchema) GetLastUsedAtField() string {
	return s.GetField(APIKeySchemaLastUsedAtField)
}

func (s *apiKeySchema) GetMetadataField() string {
	return s.GetField(APIKeySchemaMetadataField)
}

func (s *apiKeySchema) GetCreatedAtField() string {
	return s.GetField(limen.SchemaCreatedAtField)
}

func (s *apiKeySchema) GetUpdatedAtField() string {
	return s.GetField(limen.SchemaUpdatedAtField)
}

func (s *apiKeySchema) ToStorage(data limen.Model) map[string]any {
	apiKey := data.(*ApiKey)
	payload := map[string]any{
		s.GetNameField():                  apiKey.Name,
		s.GetProfileField():               apiKey.Profile,
		s.GetCreatedByField():             apiKey.CreatedByUserID,
		s.GetPrincipalTypeField():         string(apiKey.PrincipalType),
		s.GetPrincipalIDField():           apiKey.PrincipalID,
		s.GetKeyHashField():               apiKey.KeyHash,
		s.GetPrefixField():                apiKey.Prefix,
		s.GetLast4Field():                 apiKey.Last4,
		s.GetEnabledField():               apiKey.Enabled,
		s.GetExpiresAtField():             apiKey.ExpiresAt,
		s.GetRateLimitMaxField():          apiKey.RateLimitMax,
		s.GetRateLimitWindowField():       apiKey.RateLimitWindowMS,
		s.GetRateLimitRequestCountField(): apiKey.RateLimitRequestCount,
		s.GetLastUsedAtField():            apiKey.LastUsedAt,
	}

	if apiKey.Permissions != nil {
		if json, err := json.Marshal(apiKey.Permissions); err == nil {
			payload[s.GetPermissionsField()] = string(json)
		}
	}

	if apiKey.Metadata != nil {
		if json, err := json.Marshal(apiKey.Metadata); err == nil {
			payload[s.GetMetadataField()] = string(json)
		}
	}
	return payload
}

func (s *apiKeySchema) FromStorage(data map[string]any) limen.Model {
	return &ApiKey{
		ID:                    data[s.GetIDField()],
		Name:                  limen.GetValue[string](data[s.GetNameField()]),
		Profile:               limen.GetValue[string](data[s.GetProfileField()]),
		CreatedByUserID:       limen.GetValue[any](data[s.GetCreatedByField()]),
		PrincipalType:         PrincipalType(limen.GetValue[string](data[s.GetPrincipalTypeField()])),
		PrincipalID:           limen.GetValue[any](data[s.GetPrincipalIDField()]),
		KeyHash:               limen.GetValue[string](data[s.GetKeyHashField()]),
		Prefix:                limen.GetNullableValue[string](data[s.GetPrefixField()]),
		Last4:                 limen.GetValue[string](data[s.GetLast4Field()]),
		Permissions:           limen.ParseJSONFromStorage[Permissions](data, s.GetPermissionsField()),
		Enabled:               limen.GetValue[bool](data[s.GetEnabledField()]),
		ExpiresAt:             limen.GetNullableValue[time.Time](data[s.GetExpiresAtField()]),
		RateLimitMax:          limen.GetNullableValue[int32](data[s.GetRateLimitMaxField()]),
		RateLimitWindowMS:     limen.GetNullableValue[int64](data[s.GetRateLimitWindowField()]),
		RateLimitRequestCount: limen.GetNullableValue[int32](data[s.GetRateLimitRequestCountField()]),
		LastUsedAt:            limen.GetNullableValue[time.Time](data[s.GetLastUsedAtField()]),
		Metadata:              limen.ParseJSONFromStorage[map[string]any](data, s.GetMetadataField()),
		CreatedAt:             limen.GetValue[time.Time](data[s.GetCreatedAtField()]),
		UpdatedAt:             limen.GetValue[time.Time](data[s.GetUpdatedAtField()]),
		raw:                   data,
	}
}

func buildAPIKeyTableDef(schemaConfig *limen.SchemaConfig, schema *apiKeySchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(APIKeySchemaTableName),
		APIKeySchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(APIKeySchemaNameField, limen.ColumnTypeString),
		limen.WithSchemaField(APIKeySchemaProfileField, limen.ColumnTypeString),
		limen.WithSchemaField(APIKeySchemaCreatedByField, schemaConfig.GetIDColumnType(), limen.WithNullable(true)),
		limen.WithSchemaField(APIKeySchemaPrincipalTypeField, limen.ColumnTypeString),
		limen.WithSchemaField(APIKeySchemaPrincipalIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(APIKeySchemaKeyHashField, limen.ColumnTypeText),
		limen.WithSchemaField(APIKeySchemaPrefixField, limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaField(APIKeySchemaLast4Field, limen.ColumnTypeString),
		limen.WithSchemaField(APIKeySchemaEnabledField, limen.ColumnTypeBool),
		limen.WithSchemaField(APIKeySchemaPermissionsField, limen.ColumnTypeMapStringAny, limen.WithNullable(true)),
		limen.WithSchemaField(APIKeySchemaExpiresAtField, limen.ColumnTypeTime, limen.WithNullable(true)),
		limen.WithSchemaField(APIKeySchemaRateLimitMaxField, limen.ColumnTypeInt, limen.WithNullable(true)),
		limen.WithSchemaField(APIKeySchemaRateLimitWindowField, limen.ColumnTypeInt64, limen.WithNullable(true)),
		limen.WithSchemaField(APIKeySchemaRateLimitRequestCountField, limen.ColumnTypeInt, limen.WithNullable(true)),
		limen.WithSchemaField(APIKeySchemaLastUsedAtField, limen.ColumnTypeTime, limen.WithNullable(true)),
		limen.WithSchemaField(APIKeySchemaMetadataField, limen.ColumnTypeMapStringAny, limen.WithNullable(true)),
		limen.WithSchemaCreatedAtField(),
		limen.WithSchemaUpdatedAtField(),

		limen.WithSchemaUniqueIndex("idx_api_keys_key_hash", []limen.SchemaField{APIKeySchemaKeyHashField}),
		limen.WithSchemaIndex("idx_api_keys_principal", []limen.SchemaField{
			APIKeySchemaPrincipalTypeField,
			APIKeySchemaPrincipalIDField,
		}),

		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_api_keys_created_by",
			Column:           APIKeySchemaCreatedByField,
			ReferencedSchema: limen.CoreSchemaUsers,
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionSetNull,
			OnUpdate:         limen.FKActionCascade,
		}),
	)
}
