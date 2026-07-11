package apikey

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/thecodearcher/limen"
)

type apiKeyPlugin struct {
	config             *config
	db                 *limen.DatabaseActionHelper
	core               *limen.LimenCore
	apiKeySchema       *apiKeySchema
	principalResolvers map[PrincipalType]PrincipalResolver
}

func New(opts ...ConfigOption) *apiKeyPlugin {
	config := &config{
		profiles: map[string]Profile{
			"default": defaultProfile(),
		},
		keyLength: 64,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &apiKeyPlugin{
		config: config,
		principalResolvers: map[PrincipalType]PrincipalResolver{
			PrincipalTypeUser: &userPrincipalResolver{},
		},
	}
}

func (p *apiKeyPlugin) Name() limen.PluginName {
	return limen.PluginAPIKey
}

func (p *apiKeyPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	p.apiKeySchema = newAPIKeySchema()
	apiKeyTableDef := buildAPIKeyTableDef(schema, p.apiKeySchema)

	return []limen.SchemaIntrospector{apiKeyTableDef}
}

func (p *apiKeyPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/api-keys",
	}
}

func (p *apiKeyPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	handlers := newApiKeyHandlers(p, httpCore)
	routeBuilder.ProtectedPOST("/", "api-key-create", handlers.Create)
	routeBuilder.ProtectedGET("/", "api-key-list", handlers.List)
	routeBuilder.ProtectedPATCH("/:id", "api-key-update", handlers.Update)
	routeBuilder.ProtectedDELETE("/:id", "api-key-revoke", handlers.Revoke)
}

func (p *apiKeyPlugin) Initialize(core *limen.LimenCore) error {
	p.db = core.DBAction
	p.core = core

	return nil
}

func (p *apiKeyPlugin) ProfileIDs() []string {
	return slices.Collect(maps.Keys(p.config.profiles))
}

func (p *apiKeyPlugin) RegisterPrincipalResolver(principalType PrincipalType, r PrincipalResolver) {
	p.principalResolvers[principalType] = r
}

func (p *apiKeyPlugin) resolvePrincipalID(ctx context.Context, principalType PrincipalType, userID any) (principalID any, err error) {
	resolver, ok := p.principalResolvers[principalType]
	fmt.Printf("principalResolvers: %v, principalType: %v, resolver: %v, ok: %v\n", p.principalResolvers, principalType, resolver, ok)
	if !ok {
		return nil, limen.NewLimenError("principal resolver not found", http.StatusInternalServerError, nil)
	}
	return resolver.ResolvePrincipalID(ctx, string(principalType), userID)
}

func (p *apiKeyPlugin) grantablePrincipalPermissions(ctx context.Context, principalType PrincipalType, userID any) (Permissions, error) {
	resolver, ok := p.principalResolvers[principalType]
	if !ok {
		return nil, limen.NewLimenError("principal resolver not found", http.StatusInternalServerError, nil)
	}
	return resolver.GrantablePermissions(ctx, string(principalType), userID)
}

func (p *apiKeyPlugin) GetProfile(id string) (*Profile, error) {
	profile, ok := p.config.profiles[id]
	if !ok {
		return nil, limen.NewLimenError("profile not found", http.StatusNotFound, nil)
	}
	return &profile, nil
}
