package apikey

import (
	"context"
	"net/http"

	"github.com/thecodearcher/limen"
)

type PrincipalResolver interface {
	// ResolvePrincipalID resolves the principal ID for the given principal type and user ID
	ResolvePrincipalID(ctx context.Context, principalType string, userID any) (principalID any, err error)

	// GrantablePermissions resolves the grantable permissions for the given principal ID
	// The returned map is a map of resource type to a list of grantable permissions for that resource type
	// e.g. {"resource-type": ["permission-1", "permission-2"]}
	// if the map is empty, all permissions are grantable
	GrantablePermissions(ctx context.Context, principalType string, principalID any) (map[string][]string, error)
}

type PrincipalResolverRegistry interface {
	RegisterPrincipalResolver(principalType string, r PrincipalResolver)
}

type userPrincipalResolver struct {
}

func (r *userPrincipalResolver) ResolvePrincipalID(ctx context.Context, principalType string, userID any) (principalID any, err error) {
	if principalType != string(PrincipalTypeUser) {
		return nil, limen.NewLimenError("principal type not supported", http.StatusBadRequest, nil)
	}
	return userID, nil
}

func (r *userPrincipalResolver) GrantablePermissions(ctx context.Context, principalType string, principalID any) (map[string][]string, error) {
	return nil, nil
}
