package organization

import (
	"context"

	"github.com/thecodearcher/limen"
)

// ApiKeyPrincipalType binds an api-key profile's keys to the caller's active organization.
const ApiKeyPrincipalType = "organizations"

type ApiKeyPrincipal struct {
	plugin *organizationPlugin
}

// ApiKeyPrincipal returns the resolver to register under ApiKeyPrincipalType.
func (o *organizationPlugin) ApiKeyPrincipal() *ApiKeyPrincipal {
	return &ApiKeyPrincipal{plugin: o}
}

func (r *ApiKeyPrincipal) ResolvePrincipalID(ctx context.Context, _ any) (any, error) {
	session, err := limen.GetCurrentSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	// The session may hold the public id; keys store the internal one.
	organization, err := r.plugin.resolveActiveOrganization(ctx, session.Session)
	if err != nil {
		return nil, err
	}
	return organization.ID, nil
}

func (r *ApiKeyPrincipal) HasPermission(ctx context.Context, principalID any, permissions map[string][]string) error {
	session, err := limen.GetCurrentSessionFromCtx(ctx)
	if err != nil {
		return err
	}
	return r.plugin.HasPermission(ctx, session.User, principalID, permissions)
}

func (r *ApiKeyPrincipal) GrantablePermissions(ctx context.Context, principalID any) (map[string][]string, error) {
	session, err := limen.GetCurrentSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	actor, err := r.plugin.loadMemberAccess(ctx, principalID, session.User.ID)
	if err != nil {
		return nil, err
	}
	return actor.permissions, nil
}

func (r *ApiKeyPrincipal) EnsurePrincipalExists(ctx context.Context, principalID any) error {
	_, err := r.plugin.GetOrganization(ctx, principalID)
	return err
}
