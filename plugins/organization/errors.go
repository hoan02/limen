package organization

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrNoActiveOrganization = limen.NewLimenError("No active organization", http.StatusUnauthorized, nil)

	ErrOrganizationSlugAlreadyExists  = limen.NewLimenError("Organization slug not available", http.StatusConflict, nil)
	ErrInvalidSlug                    = limen.NewLimenError("The slug must contain letters or numbers", http.StatusBadRequest, nil)
	ErrMemberAlreadyExists            = limen.NewLimenError("You are already a member of this organization", http.StatusConflict, nil)
	ErrMemberRoleAlreadyExists        = limen.NewLimenError("Role already assigned to this member", http.StatusConflict, nil)
	ErrOrganizationCreationNotAllowed = limen.NewLimenError("You are not allowed to create organizations", http.StatusForbidden, nil)
	ErrUserHasReachedMaxOrganizations = limen.NewLimenError("You have reached the maximum number of organizations", http.StatusForbidden, nil)

	ErrOwnerRoleNotFound      = limen.NewLimenError("Could not find the configured owner role", http.StatusInternalServerError, nil)
	ErrInsufficientPermission = limen.NewLimenError("You do not have permission to access this resource", http.StatusForbidden, nil)

	ErrMemberNotInOrganization = limen.NewLimenError("The member is not a member of this organization", http.StatusForbidden, nil)
	ErrFailedToResolveRoles    = limen.NewLimenError("Invalid role provided", http.StatusBadRequest, nil)

	ErrUserCannotInviteOwner = limen.NewLimenError("You cannot invite the owner role. Only existing owners can invite a new owner.", http.StatusForbidden, nil)

	ErrInvitationAlreadyExists = limen.NewLimenError("Invitation already exists", http.StatusConflict, nil)
	ErrInvitationEmailMismatch = limen.NewLimenError("This invitation was sent to a different email address", http.StatusForbidden, nil)
	ErrInvalidInvitation       = limen.NewLimenError("This invitation is no longer valid", http.StatusForbidden, nil)

	ErrUserAlreadyInOrganization        = limen.NewLimenError("This email address is already in use in this organization", http.StatusConflict, nil)
	ErrMaxMembersPerOrganizationReached = limen.NewLimenError("The organization has reached the maximum number of members", http.StatusForbidden, nil)

	ErrMemberMustHaveAtLeastOneRole = limen.NewLimenError("The member must have at least one role", http.StatusForbidden, nil)
	ErrCannotRemoveLastOwner        = limen.NewLimenError("Organization must have at least one owner", http.StatusForbidden, nil)
	ErrUserCannotManageOwnerRole    = limen.NewLimenError("Only organization owners can manage the owner role", http.StatusForbidden, nil)

	ErrRoleNameReserved             = limen.NewLimenError("This role name is reserved and cannot be used", http.StatusConflict, nil)
	ErrRoleNameAlreadyExists        = limen.NewLimenError("A role with this name already exists in this organization", http.StatusConflict, nil)
	ErrRolePermissionsExceedGranted = limen.NewLimenError("You cannot grant permissions you do not hold", http.StatusForbidden, nil)
	ErrRoleStillAssignedToMembers   = limen.NewLimenError("This role is still assigned to members", http.StatusConflict, nil)
	ErrRolePermissionsCannotBeEmpty = limen.NewLimenError("The role must grant at least one permission", http.StatusBadRequest, nil)
	ErrRoleNameCannotBeEmpty        = limen.NewLimenError("The role name is required", http.StatusBadRequest, nil)
	ErrOrganizationRoleNotFound     = limen.NewLimenError("Role not found in this organization", http.StatusNotFound, nil)

	ErrCustomRolesDisabled            = limen.NewLimenError("Custom roles are not enabled", http.StatusNotFound, nil)
	ErrMaxRolesPerOrganizationReached = limen.NewLimenError("The organization has reached the maximum number of roles", http.StatusForbidden, nil)
)
