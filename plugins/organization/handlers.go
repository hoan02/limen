package organization

import (
	"net/http"
	"time"

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
		RateLimitRules: []*limen.RateLimitRule{
			limen.NewRateLimitRuleWithMethod(http.MethodPost, "/", 5, 10*time.Second),
			limen.NewRateLimitRule("/invitations/respond", 5, 10*time.Second),
			limen.NewRateLimitRuleWithMethod(http.MethodPost, "/invitations", 5, 10*time.Second),
			limen.NewRateLimitRuleWithMethod(http.MethodDelete, "/invitations/:id", 5, 10*time.Second),
		},
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

	routeBuilder.ProtectedGET("/members", "organizations:members-list", h.ListMembers,
		h.plugin.HasPermissionMiddleware(perms("organization:read", "member:read")),
	)
	routeBuilder.ProtectedGET("/me", "organizations:member-get", h.GetMember, h.plugin.CanAccessOrganizationMiddleware())
	routeBuilder.ProtectedPOST("/switch", "organizations:switch", h.SwitchOrganization)

	routeBuilder.ProtectedPOST("/invitations", "organizations:invite-member", h.InviteMember,
		h.plugin.RequireActiveOrganizationMiddleware(),
		h.plugin.HasPermissionMiddleware(perms("invitation:create")),
	)
	routeBuilder.ProtectedPOST("/invitations/respond", "organizations:respond-to-invitation", h.RespondToInvitation)

	routeBuilder.ProtectedDELETE("/invitations/:id", "organizations:cancel-pending-invitation", h.CancelPendingInvitation,
		h.plugin.RequireActiveOrganizationMiddleware(),
		h.plugin.HasPermissionMiddleware(perms("invitation:cancel")),
	)
	routeBuilder.ProtectedGET("/invitations", "organizations:list-invitations", h.ListInvitations,
		h.plugin.RequireActiveOrganizationMiddleware(),
		h.plugin.HasPermissionMiddleware(perms("invitation:read")),
	)
}

func (h *organizationHandlers) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[CreateOrganizationRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("name").Required().String()
		v.Field("slug").Required().String()
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

	if _, err := h.plugin.SetActiveOrganization(r.Context(), session.Session, organization.ID); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.AddHeader(w, HeaderActiveOrganizationID, h.plugin.clientOrganizationID(organization))
	h.responder.JSON(w, r, http.StatusCreated, h.plugin.core.SerializeModel(h.plugin.organizationSchema, organization))
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

func (h *organizationHandlers) GetMember(w http.ResponseWriter, r *http.Request) {
	session, organizationID, err := GetActiveOrganizationIDFromCtx(r.Context())
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
	_, organizationID, err := GetActiveOrganizationIDFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	members, err := h.plugin.ListMembersWithRelations(r.Context(), organizationID, limen.ParsePagination(r))
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
		v.Field("organization").Required().String()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	organizationID := body["organization"].(string)

	organization, err := h.plugin.SwitchOrganization(r.Context(), session.Session, organizationID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.AddHeader(w, HeaderActiveOrganizationID, h.plugin.clientOrganizationID(organization))
	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.organizationSchema, organization))
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

func (h *organizationHandlers) CancelPendingInvitation(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, h.responder, func(v *limen.Validator) {
		v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
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

	invitation, err := h.plugin.CancelPendingInvitation(r.Context(), session.Organization, limen.GetParam(r, "id"))
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

	invitations, err := h.plugin.ListInvitations(r.Context(), session.Organization, options)
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
