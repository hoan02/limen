package apikey

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/thecodearcher/limen"
)

type apiKeyPlugin struct {
	config             *config
	store              *apiKeyStore
	db                 *limen.DatabaseActionHelper
	core               *limen.LimenCore
	apiKeySchema       *apiKeySchema
	principalResolvers map[PrincipalType]PrincipalResolver
	rateLimiter        rateLimiter
}

func New(opts ...ConfigOption) *apiKeyPlugin {
	config := &config{
		profiles: map[string]Profile{
			"default": defaultProfile(),
		},
		cacheEnabled:       true,
		cacheTTL:           5 * time.Minute,
		rateLimitStoreType: limen.StoreTypeCache,
		lastUsedAtThrottle: 5 * time.Minute,
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
	p.apiKeySchema = newAPIKeySchema(p.config)
	apiKeyTableDef := buildAPIKeyTableDef(schema, p.apiKeySchema)

	return []limen.SchemaIntrospector{apiKeyTableDef}
}

func (p *apiKeyPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/api-keys",
		RateLimitRules: []*limen.RateLimitRule{
			limen.NewRateLimitRuleWithMethod("/", http.MethodPost, 5, 10*time.Second),
			limen.NewRateLimitRule("/:id/rotate", 5, 10*time.Second),
		},
	}
}

func (p *apiKeyPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	handlers := newApiKeyHandlers(p, httpCore)
	routeBuilder.ProtectedPOST("/", "api-key-create", handlers.Create)
	routeBuilder.ProtectedGET("/:id", "api-key-get", handlers.Get)
	routeBuilder.ProtectedGET("/", "api-key-list", handlers.List)
	routeBuilder.ProtectedPATCH("/:id", "api-key-update", handlers.Update)
	routeBuilder.ProtectedDELETE("/:id", "api-key-revoke", handlers.Revoke)
	routeBuilder.ProtectedPOST("/:id/rotate", "api-key-rotate", handlers.Rotate)
}

func (p *apiKeyPlugin) Initialize(core *limen.LimenCore) error {
	p.db = core.DBAction
	p.core = core
	p.rateLimiter = resolveRateLimiter(p)
	p.store = newApiKeyStore(p)
	return nil
}

func (p *apiKeyPlugin) ProfileIDs() []string {
	return slices.Collect(maps.Keys(p.config.profiles))
}

func (p *apiKeyPlugin) RegisterPrincipalResolver(principalType string, r PrincipalResolver) {
	p.principalResolvers[PrincipalType(principalType)] = r
}

func (p *apiKeyPlugin) resolvePrincipalID(ctx context.Context, principalType PrincipalType, userID any) (principalID any, err error) {
	resolver, ok := p.principalResolvers[principalType]
	if !ok {
		return nil, limen.NewLimenError(fmt.Sprintf("principal resolver not found for type %s", principalType), http.StatusInternalServerError, nil)
	}
	return resolver.ResolvePrincipalID(ctx, string(principalType), userID)
}

func (p *apiKeyPlugin) grantablePrincipalPermissions(ctx context.Context, principalType PrincipalType, principalID any) (Permissions, error) {
	resolver, ok := p.principalResolvers[principalType]
	if !ok {
		return nil, limen.NewLimenError("principal resolver not found", http.StatusInternalServerError, nil)
	}
	return resolver.GrantablePermissions(ctx, string(principalType), principalID)
}

func (p *apiKeyPlugin) GetProfile(id string) (*Profile, error) {
	profile, ok := p.config.profiles[id]
	if !ok {
		return nil, limen.NewLimenError("profile not found", http.StatusNotFound, nil)
	}
	return &profile, nil
}

func resolveRateLimiter(plugin *apiKeyPlugin) rateLimiter {
	if plugin.config.rateLimitStoreType == limen.StoreTypeCache {
		return newCacheRateLimiter(plugin)
	}
	return newDatabaseRateLimiter(plugin)
}
