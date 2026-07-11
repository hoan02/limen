package apikey

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

type apiKeyHandlers struct {
	plugin    *apiKeyPlugin
	httpCore  *limen.LimenHTTPCore
	responder *limen.Responder
	builder   *limen.RouteBuilder
}

func newApiKeyHandlers(plugin *apiKeyPlugin, httpCore *limen.LimenHTTPCore) *apiKeyHandlers {
	return &apiKeyHandlers{
		plugin:    plugin,
		httpCore:  httpCore,
		responder: httpCore.Responder,
	}
}

func (h *apiKeyHandlers) Create(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[ApiKeyCreateRequest](w, r, h.responder, func(v *limen.Validator) {
		profileIDs := h.plugin.ProfileIDs()

		v.Field("profile").Optional(defaultProfile().ID).String().In(profileIDs)
		v.Field("name").Required().String()
		v.Field("permissions").Optional().Object()
		v.Field("expires_in").Optional().Number().Min(5 * 60).Max(3600)
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	result, err := h.plugin.Create(r.Context(), session.User, body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusCreated, result)
}

func (h *apiKeyHandlers) List(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	page, err := h.plugin.List(r.Context(), session.User, r.URL.Query().Get("profile"), r.URL.Query().Get("enabled") == "true", limen.ParsePagination(r))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, page)
}

func (h *apiKeyHandlers) Update(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[ApiKeyUpdateRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("name").Optional().String()
		v.Field("permissions").Optional().Object()
		v.Field("all_permissions").Optional().Boolean()
		v.Field("enabled").Optional().Boolean()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	result, err := h.plugin.Update(r.Context(), session.User, limen.GetParam(r, "id"), body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, result)
}

func (h *apiKeyHandlers) Revoke(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	err = h.plugin.Revoke(r.Context(), session.User, limen.GetParam(r, "id"), r.URL.Query().Get("is_temporary") == "true")
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}
