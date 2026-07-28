package organization

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrNoActiveOrganization = limen.NewLimenError("No active organization", http.StatusUnauthorized, nil)

	ErrOrganizationSlugAlreadyExists  = limen.NewLimenError("Organization slug not available", http.StatusConflict, nil)
	ErrMemberAlreadyExists            = limen.NewLimenError("You are already a member of this organization", http.StatusConflict, nil)
	ErrMemberRoleAlreadyExists        = limen.NewLimenError("Role already assigned to this member", http.StatusConflict, nil)
	ErrOrganizationCreationNotAllowed = limen.NewLimenError("You are not allowed to create organizations", http.StatusForbidden, nil)
	ErrUserHasReachedMaxOrganizations = limen.NewLimenError("You have reached the maximum number of organizations", http.StatusForbidden, nil)

	ErrOwnerRoleNotFound      = limen.NewLimenError("Could not find the configured owner role", http.StatusInternalServerError, nil)
	ErrInsufficientPermission = limen.NewLimenError("You do not have permission to access this resource", http.StatusForbidden, nil)

	ErrMemberNotInOrganization = limen.NewLimenError("You are not a member of this organization", http.StatusForbidden, nil)
)
