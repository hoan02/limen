package organization

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

type organizationHandlers struct {
	plugin    *organizationPlugin
	responder *limen.Responder
}

func NewOrganizationHandlers(plugin *organizationPlugin, httpCore *limen.LimenHTTPCore) *organizationHandlers {
	return &organizationHandlers{plugin: plugin, responder: httpCore.Responder}
}

func (p *organizationPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/organizations",
	}
}

func (p *organizationPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	api := NewOrganizationHandlers(p, httpCore)
	routes(api, routeBuilder)
	p.responder = httpCore.Responder
}

func routes(h *organizationHandlers, routeBuilder *limen.RouteBuilder) {
	routeBuilder.ProtectedPOST("/", "organizations:create", h.CreateOrganization)
	routeBuilder.ProtectedGET("/", "organizations:list", h.ListOrganizations)
	routeBuilder.ProtectedPOST("/check-slug", "organizations:check-slug", h.CheckSlugAvailability)
	routeBuilder.ProtectedPATCH("/:id", "organizations:update", h.UpdateOrganization)
	routeBuilder.ProtectedDELETE("/:id", "organizations:delete", h.DeleteOrganization)

	routeBuilder.ProtectedGET("/members", "organizations:members-list", h.ListMembers)
	routeBuilder.ProtectedGET("/me", "organizations:member-get", h.GetMember)
	routeBuilder.ProtectedGET("/active", "organizations:get-active", h.GetActiveOrganization)
	routeBuilder.ProtectedPOST("/switch", "organizations:switch", h.SwitchOrganization)
	routeBuilder.ProtectedPOST("/leave", "organizations:leave-organization", h.LeaveOrganization)

	routeBuilder.ProtectedPOST("/invitations", "organizations:invite-member", h.InviteMember, h.plugin.RequireActiveOrganizationMiddleware())
	routeBuilder.ProtectedPOST("/invitations/respond", "organizations:respond-to-invitation", h.RespondToInvitation)
	routeBuilder.ProtectedGET("/invitations/token/:token", "organizations:get-invitation-by-token", h.GetInvitationByToken)
	routeBuilder.ProtectedPOST("/invitations/cancel", "organizations:cancel-pending-invitation", h.CancelPendingInvitation, h.plugin.RequireActiveOrganizationMiddleware())
	routeBuilder.ProtectedGET("/invitations", "organizations:list-invitations", h.ListInvitations, h.plugin.RequireActiveOrganizationMiddleware())

	routeBuilder.ProtectedPOST("/members/:id/roles/revoke", "organizations:revoke-member-role", h.RevokeMemberRoles, h.plugin.RequireActiveOrganizationMiddleware())
	routeBuilder.ProtectedPOST("/members/:id/roles/assign", "organizations:assign-member-role", h.AssignMemberRoles, h.plugin.RequireActiveOrganizationMiddleware())
	routeBuilder.ProtectedDELETE("/members/:id", "organizations:remove-member", h.RemoveMember, h.plugin.RequireActiveOrganizationMiddleware())

	if h.plugin.config.customRolesEnabled {
		routeBuilder.ProtectedPOST("/roles", "organizations:create-role", h.CreateOrganizationRole, h.plugin.RequireActiveOrganizationMiddleware())
		routeBuilder.ProtectedGET("/roles", "organizations:list-roles", h.ListOrganizationRoles, h.plugin.RequireActiveOrganizationMiddleware())
		routeBuilder.ProtectedPATCH("/roles/:id", "organizations:update-role", h.UpdateOrganizationRole, h.plugin.RequireActiveOrganizationMiddleware())
		routeBuilder.ProtectedDELETE("/roles/:id", "organizations:delete-role", h.DeleteOrganizationRole, h.plugin.RequireActiveOrganizationMiddleware())
	}
}

func (h *organizationHandlers) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[CreateOrganizationRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("name").Required().String()
		v.Field("slug").Optional().String()
		v.Field("logo").Optional().String()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	organization, err := h.plugin.CreateOrganization(r.Context(), session.User, body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	sessionResult, err := h.plugin.SetActiveOrganization(r.Context(), session.Session, organization)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.AddHeader(w, HeaderActiveOrganizationID, h.plugin.clientOrganizationID(organization))
	h.responder.JSONWithSession(w, r, http.StatusCreated, h.plugin.core.SerializeModel(h.plugin.organizationSchema, organization), sessionResult)
}

func (h *organizationHandlers) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[UpdateOrganizationRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
			return limen.ValidateClientIDValue(h.plugin.core, h.plugin.organizationSchema, value)
		})
		v.Field("name").Optional().String()
		v.Field("slug").Optional().String()
		v.Field("logo").Optional().String()
		v.Field("metadata").Optional().Object()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	organization, err := h.plugin.UpdateOrganization(r.Context(), session.User, limen.GetParam(r, "id"), body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.organizationSchema, organization))
}

func (h *organizationHandlers) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
			return limen.ValidateClientIDValue(h.plugin.core, h.plugin.organizationSchema, value)
		})
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	if err := h.plugin.DeleteOrganization(r.Context(), session.User, limen.GetParam(r, "id")); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *organizationHandlers) CheckSlugAvailability(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Field("slug").Required().String()
	})

	if body == nil {
		return
	}

	available, err := h.plugin.CheckSlugAvailability(r.Context(), body["slug"].(string))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, map[string]any{
		"available": available,
		"slug":      h.plugin.applySlugNormalization(body["slug"].(string)),
	})
}

func (h *organizationHandlers) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	filter := limen.BindAndValidate[ListOrganizationsFilter](w, r, h.responder, func(v *limen.Validator) {
		v.Field("name").Optional().String()
	})

	if filter == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	data, err := h.plugin.ListOrganizations(r.Context(), session.User, filter, limen.ParsePagination(r))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, &limen.Page[map[string]any]{
		Items:      limen.SerializeModels(h.plugin.core, h.plugin.organizationSchema, data.Items),
		Total:      data.Total,
		Page:       data.Page,
		PerPage:    data.PerPage,
		TotalPages: data.TotalPages,
	})
}

func (h *organizationHandlers) GetActiveOrganization(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	organization, err := h.plugin.GetActiveOrganization(r.Context(), session.Session, session.User)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.organizationSchema, organization))
}

func (h *organizationHandlers) GetMember(w http.ResponseWriter, r *http.Request) {
	session, organizationID, err := h.plugin.GetActiveOrganizationIDFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	member, err := h.plugin.GetMemberWithRelations(r.Context(), session.User, organizationID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.memberSchema, member))
}

func (h *organizationHandlers) ListMembers(w http.ResponseWriter, r *http.Request) {
	session, organizationID, err := h.plugin.GetActiveOrganizationIDFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	members, err := h.plugin.ListMembersWithRelations(r.Context(), session.User, organizationID, limen.ParsePagination(r))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, &limen.Page[map[string]any]{
		Items:      limen.SerializeModels(h.plugin.core, h.plugin.memberSchema, members.Items),
		Total:      members.Total,
		Page:       members.Page,
		PerPage:    members.PerPage,
		TotalPages: members.TotalPages,
	})
}

func (h *organizationHandlers) SwitchOrganization(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Field("organization").Nullable().String().Custom(func(value any, _ map[string]any) error {
			return limen.ValidateClientIDValue(h.plugin.core, h.plugin.organizationSchema, value)
		})
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	organization, sessionResult, err := h.plugin.SwitchOrganization(r.Context(), session.Session, body["organization"])
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	if organization == nil {
		h.responder.JSONWithSession(w, r, http.StatusOK, nil, sessionResult)
		return
	}

	h.responder.AddHeader(w, HeaderActiveOrganizationID, h.plugin.clientOrganizationID(organization))
	h.responder.JSONWithSession(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.organizationSchema, organization), sessionResult)
}

func (h *organizationHandlers) InviteMember(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[CreateInvitationRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("email").Required().Email()
		v.Field("role").Required().String()
		v.Field("resend").Optional().Boolean()
	})

	if body == nil {
		return
	}

	session, err := GetActiveOrganizationSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	invitation, err := h.plugin.CreateInvitation(r.Context(), session.Session.User, session.Organization, body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.invitationSchema, invitation))
}

func (h *organizationHandlers) RespondToInvitation(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Field("response").Required().String()
		v.Field("token").Required().String()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	invitation, err := h.plugin.RespondToInvitation(r.Context(), session.User, body["token"].(string), InvitationResponse(body["response"].(string)))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.invitationSchema, invitation))
}

func (h *organizationHandlers) GetInvitationByToken(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Param("token").Required().String()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	invitation, err := h.plugin.GetInvitationByToken(r.Context(), session.User, limen.GetParam(r, "token"))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.invitationSchema, invitation))
}

func (h *organizationHandlers) CancelPendingInvitation(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Field("invitation").Required().String().Custom(func(value any, _ map[string]any) error {
			return limen.ValidateClientIDValue(h.plugin.core, h.plugin.invitationSchema, value)
		})
	})

	if body == nil {
		return
	}

	session, err := GetActiveOrganizationSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	invitation, err := h.plugin.CancelPendingInvitation(r.Context(), session.Session.User, session.Organization, body["invitation"].(string))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.invitationSchema, invitation))
}

func (h *organizationHandlers) ListInvitations(w http.ResponseWriter, r *http.Request) {
	filter := limen.BindAndValidate[ListInvitationsOptions](w, r, h.responder, func(v *limen.Validator) {
		v.Field("statuses").Optional().Array().String().In([]string{
			string(InvitationStatusPending),
			string(InvitationStatusAccepted),
			string(InvitationStatusRejected),
			string(InvitationStatusCanceled),
		})
	})

	if filter == nil {
		return
	}

	options := &ListInvitationsOptions{
		QueryOptions: limen.ParsePagination(r),
		Statuses:     filter.Statuses,
	}

	session, err := GetActiveOrganizationSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	invitations, err := h.plugin.ListInvitationsWithRelations(r.Context(), session.Session.User, session.Organization, options)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, &limen.Page[map[string]any]{
		Items:      limen.SerializeModels(h.plugin.core, h.plugin.invitationSchema, invitations.Items),
		Total:      invitations.Total,
		Page:       invitations.Page,
		PerPage:    invitations.PerPage,
		TotalPages: invitations.TotalPages,
	})
}

func (h *organizationHandlers) AssignMemberRoles(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
			return limen.ValidateClientIDValue(h.plugin.core, h.plugin.memberSchema, value)
		})
		v.Field("roles").Required().Array().String().Length(1)
	})

	if body == nil {
		return
	}

	session, err := GetActiveOrganizationSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	err = h.plugin.AssignMemberRole(r.Context(), session.Session.User, session.Organization, limen.GetParam(r, "id"), body["roles"].([]string)[0])
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *organizationHandlers) RevokeMemberRoles(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
			return limen.ValidateClientIDValue(h.plugin.core, h.plugin.memberSchema, value)
		})
		v.Field("roles").Required().Array().String().Length(1)
	})

	if body == nil {
		return
	}

	session, err := GetActiveOrganizationSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	role := body["roles"].([]string)[0]
	err = h.plugin.RevokeMemberRole(r.Context(), session.Session.User, session.Organization, limen.GetParam(r, "id"), role)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *organizationHandlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
			return limen.ValidateClientIDValue(h.plugin.core, h.plugin.memberSchema, value)
		})
	})

	if body == nil {
		return
	}

	session, err := GetActiveOrganizationSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	err = h.plugin.RemoveMember(r.Context(), session.Session.User, session.Organization, limen.GetParam(r, "id"))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *organizationHandlers) LeaveOrganization(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Field("organization").Required().String().Custom(func(value any, _ map[string]any) error {
			return limen.ValidateClientIDValue(h.plugin.core, h.plugin.organizationSchema, value)
		})
	})

	if body == nil {
		return
	}
	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	sessionResult, err := h.plugin.LeaveOrganization(r.Context(), session.Session, body["organization"].(string))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSONWithSession(w, r, http.StatusNoContent, nil, sessionResult)
}

func (h *organizationHandlers) CreateOrganizationRole(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[CreateOrganizationRoleRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("name").Required().String().MaxLength(64)
		v.Field("description").Optional().String().MaxLength(255)
		v.Field("permissions").Required().Object()
	})

	if body == nil {
		return
	}

	session, err := GetActiveOrganizationSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	role, err := h.plugin.CreateOrganizationRole(r.Context(), session.Session.User, session.Organization, body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusCreated, h.plugin.core.SerializeModel(h.plugin.organizationRoleSchema, role))
}

func (h *organizationHandlers) ListOrganizationRoles(w http.ResponseWriter, r *http.Request) {
	session, err := GetActiveOrganizationSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	roles, err := h.plugin.ListOrganizationRoles(r.Context(), session.Session.User, session.Organization, limen.ParsePagination(r))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, &limen.Page[map[string]any]{
		Items:      limen.SerializeModels(h.plugin.core, h.plugin.organizationRoleSchema, roles.Items),
		Total:      roles.Total,
		Page:       roles.Page,
		PerPage:    roles.PerPage,
		TotalPages: roles.TotalPages,
	})
}

func (h *organizationHandlers) UpdateOrganizationRole(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[UpdateOrganizationRoleRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
			return limen.ValidateClientIDValue(h.plugin.core, h.plugin.organizationRoleSchema, value)
		})
		v.Field("description").Optional().String().MaxLength(255)
		v.Field("permissions").Optional().Object()
	})

	if body == nil {
		return
	}

	session, err := GetActiveOrganizationSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	role, err := h.plugin.UpdateOrganizationRole(r.Context(), session.Session.User, session.Organization, limen.GetParam(r, "id"), body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.organizationRoleSchema, role))
}

func (h *organizationHandlers) DeleteOrganizationRole(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
			return limen.ValidateClientIDValue(h.plugin.core, h.plugin.organizationRoleSchema, value)
		})
	})

	if body == nil {
		return
	}

	session, err := GetActiveOrganizationSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	if err := h.plugin.DeleteOrganizationRole(r.Context(), session.Session.User, session.Organization, limen.GetParam(r, "id")); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}
