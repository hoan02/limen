package organization

import (
	"net/http"
	"time"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
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

	routeBuilder.ProtectedGET("/members", "organizations:members-list", h.ListMembers, h.plugin.HasPermissionMiddleware(access.Permissions{
		"organization": {"read"},
		"member":       {"read"},
	}))
	routeBuilder.ProtectedGET("/me", "organizations:member-get", h.GetMember, h.plugin.CanAccessOrganizationMiddleware())
	routeBuilder.ProtectedPOST("/switch", "organizations:switch", h.SwitchOrganization, h.plugin.CanAccessOrganizationMiddleware())
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
	session, organizationID, err := CurrentOrganizationIDFromCtx(r.Context())
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
	_, organizationID, err := CurrentOrganizationIDFromCtx(r.Context())
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
