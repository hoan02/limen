package apikey

import "context"

var _ PrincipalResolverRegistry = (*apiKeyPlugin)(nil)

type PrincipalResolver interface {
	ResolvePrincipalID(ctx context.Context, userID any) (principalID any, err error)

	HasPermission(ctx context.Context, principalID any, permissions map[string][]string) error

	// GrantablePermissions resolves the grantable permissions for the given principal ID
	// The returned map is a map of resource type to a list of grantable permissions for that resource type
	// e.g. {"resource-type": ["permission-1", "permission-2"]}
	// if the map is empty, all permissions are grantable
	GrantablePermissions(ctx context.Context, principalID any) (map[string][]string, error)

	// Must report limen.ErrRecordNotFound when the principal is gone.
	EnsurePrincipalExists(ctx context.Context, principalID any) error
}

type PrincipalResolverRegistry interface {
	RegisterPrincipalResolver(principalType string, r PrincipalResolver)
}

// userPrincipalResolver governs keys a user owns outright, so nothing to check.
type userPrincipalResolver struct {
}

func (r *userPrincipalResolver) ResolvePrincipalID(ctx context.Context, userID any) (principalID any, err error) {
	return userID, nil
}

func (r *userPrincipalResolver) HasPermission(ctx context.Context, principalID any, permissions map[string][]string) error {
	return nil
}

func (r *userPrincipalResolver) GrantablePermissions(ctx context.Context, principalID any) (map[string][]string, error) {
	return nil, nil
}

func (r *userPrincipalResolver) EnsurePrincipalExists(ctx context.Context, principalID any) error {
	return nil
}
