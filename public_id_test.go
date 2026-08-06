package limen

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPublicIDConfig(generator PublicIDGenerator) PublicIDConfig {
	return PublicIDConfig{
		Generator: generator,
		Matcher: func(_ SchemaName, value string) bool {
			return strings.HasPrefix(value, "usr_")
		},
		Encoder: func(_ SchemaName, value string) string {
			return "usr_" + value
		},
		Decoder: func(_ SchemaName, value string) (string, error) {
			decoded, ok := strings.CutPrefix(value, "usr_")
			if !ok || decoded == "" || decoded == "bad" {
				return "", errors.New("malformed public ID")
			}
			return decoded, nil
		},
	}
}

func noopPublicIDGenerator() PublicIDGenerator {
	return func(context.Context, SchemaName) (string, error) {
		return "raw", nil
	}
}

func usersOnlyPublicIDConfig(generator PublicIDGenerator) PublicIDConfig {
	config := testPublicIDConfig(generator)
	config.DisabledFor = []SchemaName{
		CoreSchemaAccounts,
		CoreSchemaSessions,
		CoreSchemaVerifications,
		CoreSchemaRateLimits,
	}
	return config
}

func newPublicIDFixture(t *testing.T, config PublicIDConfig, schemaOpts ...SchemaConfigOption) (*Limen, Schema) {
	t.Helper()

	opts := append([]SchemaConfigOption{WithPublicIDs(config)}, schemaOpts...)
	limen, err := New(&Config{
		BaseURL:  "http://localhost:8080",
		Database: newTestMemoryAdapter(t),
		Schema:   NewDefaultSchemaConfig(opts...),
		Secret:   testSecret,
	})
	require.NoError(t, err)
	return limen, limen.core.Schema.User
}

func discoverPublicIDs(t *testing.T, config PublicIDConfig, plugins ...Plugin) map[SchemaName]SchemaDefinition {
	t.Helper()

	schemas, err := discoverSchemas(NewDefaultSchemaConfig(WithPublicIDs(config)), plugins)
	require.NoError(t, err)
	return schemas
}

func findColumn(columns []ColumnDefinition, field SchemaField) *ColumnDefinition {
	i := slices.IndexFunc(columns, func(column ColumnDefinition) bool {
		return column.LogicalField == field
	})
	if i == -1 {
		return nil
	}
	return &columns[i]
}

type publicIDOptOutRow map[string]any

func (r publicIDOptOutRow) Raw() map[string]any { return r }

type publicIDOptOutSchema struct{ BaseSchema }

func (s *publicIDOptOutSchema) ToStorage(data Model) map[string]any   { return data.Raw() }
func (s *publicIDOptOutSchema) FromStorage(data map[string]any) Model { return publicIDOptOutRow(data) }

const (
	publicIDOptOutSchemaName SchemaName      = "public_id_opt_out_schema"
	publicIDOptOutTableName  SchemaTableName = "public_id_opt_out_rows"
)

type publicIDOptOutPlugin struct{}

func (p *publicIDOptOutPlugin) Name() PluginName                             { return PluginName("public-id-opt-out-test") }
func (p *publicIDOptOutPlugin) Initialize(*LimenCore) error                  { return nil }
func (p *publicIDOptOutPlugin) PluginHTTPConfig() PluginHTTPConfig           { return PluginHTTPConfig{} }
func (p *publicIDOptOutPlugin) RegisterRoutes(*LimenHTTPCore, *RouteBuilder) {}
func (p *publicIDOptOutPlugin) GetSchemas(schema *SchemaConfig) []SchemaIntrospector {
	return []SchemaIntrospector{
		NewSchemaDefinitionForTable(
			publicIDOptOutSchemaName,
			publicIDOptOutTableName,
			&publicIDOptOutSchema{},
			WithSchemaIDField(schema),
			WithSchemaField("name", ColumnTypeString),
			WithDisablePublicID(),
		),
	}
}

func TestPublicIDs_Discovery(t *testing.T) {
	t.Parallel()

	t.Run("enabled schema", func(t *testing.T) {
		t.Parallel()

		config := usersOnlyPublicIDConfig(func(context.Context, SchemaName) (string, error) {
			return "01900000-0000-7000-8000-000000000000", nil
		})
		config.ColumnName = "uuid"
		config.ColumnType = ColumnTypeUUID

		schemas := discoverPublicIDs(t, config)
		column := findColumn(schemas[CoreSchemaUsers].Columns, SchemaPublicIDField)
		require.NotNil(t, column)
		assert.Equal(t, "uuid", column.Name)
		assert.Equal(t, ColumnTypeUUID, column.Type)
	})

	t.Run("disabled schemas", func(t *testing.T) {
		t.Parallel()

		schemas := discoverPublicIDs(t, usersOnlyPublicIDConfig(noopPublicIDGenerator()))
		for _, name := range []SchemaName{CoreSchemaSessions, CoreSchemaAccounts} {
			assert.Nil(t, findColumn(schemas[name].Columns, SchemaPublicIDField), "schema %s", name)
		}
	})

	t.Run("globally disabled", func(t *testing.T) {
		t.Parallel()

		config := testPublicIDConfig(noopPublicIDGenerator())
		config.Disabled = true
		schemas := discoverPublicIDs(t, config)

		for name, schema := range schemas {
			assert.Nil(t, findColumn(schema.Columns, SchemaPublicIDField), "schema %s", name)
		}
	})

	t.Run("plugin opt out", func(t *testing.T) {
		t.Parallel()

		schemas := discoverPublicIDs(t, usersOnlyPublicIDConfig(noopPublicIDGenerator()), &publicIDOptOutPlugin{})
		assert.Nil(t, findColumn(schemas[publicIDOptOutSchemaName].Columns, SchemaPublicIDField))
		assert.NotNil(t, findColumn(schemas[CoreSchemaUsers].Columns, SchemaPublicIDField))
	})
}

func TestPublicIDs_Runtime(t *testing.T) {
	t.Parallel()

	limen, schema := newPublicIDFixture(t, usersOnlyPublicIDConfig(func(context.Context, SchemaName) (string, error) {
		return "raw-1", nil
	}))
	ctx := context.Background()

	require.NoError(t, limen.core.Create(ctx, schema, &User{Email: "first@test.com"}, nil))
	require.NoError(t, limen.core.Create(ctx, schema, &User{Email: "second@test.com"}, map[string]any{
		schema.GetField(SchemaPublicIDField): "provided",
	}))

	user, err := limen.core.FindOne(ctx, schema, []Where{Eq(schema.GetIDField(), "usr_raw-1")}, nil)
	require.NoError(t, err)
	assert.Equal(t, "raw-1", user.Raw()[schema.GetField(SchemaPublicIDField)])

	require.NoError(t, limen.core.Update(ctx, schema, map[SchemaField]any{
		UserSchemaEmailField: "updated@test.com",
	}, []Where{Eq(schema.GetIDField(), "usr_provided")}))

	updated, err := limen.core.FindOne(ctx, schema, []Where{Eq(schema.GetIDField(), "usr_provided")}, nil)
	require.NoError(t, err)
	assert.Equal(t, "updated@test.com", updated.(*User).Email)

	require.NoError(t, limen.core.Delete(ctx, schema, []Where{Eq(schema.GetIDField(), "usr_provided")}))
	exists, err := limen.core.Exists(ctx, schema, []Where{Eq(schema.GetIDField(), "usr_provided")})
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPublicIDs_RewriteInAnySlice(t *testing.T) {
	t.Parallel()

	limen, schema := newPublicIDFixture(t, usersOnlyPublicIDConfig(func(context.Context, SchemaName) (string, error) {
		return "raw-1", nil
	}))
	ctx := t.Context()

	require.NoError(t, limen.core.Create(ctx, schema, &User{Email: "a@test.com"}, nil))
	require.NoError(t, limen.core.Create(ctx, schema, &User{Email: "b@test.com"}, map[string]any{
		schema.GetField(SchemaPublicIDField): "provided",
	}))

	// In() stores []any; rewrite must decode public IDs instead of querying the PK column.
	users, err := limen.core.FindMany(ctx, schema, []Where{
		In(schema.GetIDField(), []any{"usr_raw-1", "usr_provided"}),
	})
	require.NoError(t, err)
	require.Len(t, users, 2)

	emails := []string{users[0].(*User).Email, users[1].(*User).Email}
	assert.ElementsMatch(t, []string{"a@test.com", "b@test.com"}, emails)
}

func TestPublicIDs_DisableResponseTransform(t *testing.T) {
	t.Parallel()

	config := usersOnlyPublicIDConfig(func(context.Context, SchemaName) (string, error) {
		return "raw", nil
	})
	config.DisableResponseTransform = true
	limen, schema := newPublicIDFixture(t, config)
	ctx := context.Background()

	require.NoError(t, limen.core.Create(ctx, schema, &User{Email: "raw@test.com"}, nil))
	user, err := limen.core.FindOne(ctx, schema, []Where{Eq(schema.GetField(UserSchemaEmailField), "raw@test.com")}, nil)
	require.NoError(t, err)

	response := limen.core.SerializeModel(schema, user)
	assert.Equal(t, "raw", response[schema.GetField(SchemaPublicIDField)])
	assert.NotContains(t, response, "id")
}
