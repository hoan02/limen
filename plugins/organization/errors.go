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
	ErrFailedToResolveRoles    = limen.NewLimenError("Invalid role provided", http.StatusBadRequest, nil)

	ErrUserCannotInviteOwner = limen.NewLimenError("You cannot invite the owner role. Only existing owners can invite a new owner.", http.StatusForbidden, nil)

	ErrInvitationAlreadyExists = limen.NewLimenError("Invitation already exists", http.StatusConflict, nil)
	ErrInvitationEmailMismatch = limen.NewLimenError("This invitation was sent to a different email address", http.StatusForbidden, nil)
	ErrInvalidInvitation       = limen.NewLimenError("This invitation is no longer valid", http.StatusForbidden, nil)

	ErrUserAlreadyInOrganization        = limen.NewLimenError("This email address is already in use in this organization", http.StatusConflict, nil)
	ErrMaxMembersPerOrganizationReached = limen.NewLimenError("The organization has reached the maximum number of members", http.StatusForbidden, nil)
)
