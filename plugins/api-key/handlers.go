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

const MaxExpiresIn = 6307200000 // 200 years in seconds 😒 why would you need a key that long?

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
		v.Field("name").Required().String().MinLength(3).MaxLength(100)
		v.Field("permissions").Optional().Object()
		v.Field("expires_in").Optional().Number().Min(5 * 60).Max(MaxExpiresIn)
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
	filter := limen.BindAndValidate[ApiKeyListFilter](w, r, h.responder, func(v *limen.Validator) {
		v.Field("profile").Optional(defaultProfile().ID).String().In(h.plugin.ProfileIDs())
		v.Field("status").Optional().String().In([]string{string(APIKeyStatusEnabled), string(APIKeyStatusDisabled)})
	})

	if filter == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	page, err := h.plugin.List(r.Context(), session.User, filter, limen.ParsePagination(r))
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

	err = h.plugin.Revoke(r.Context(), session.User, limen.GetParam(r, "id"))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *apiKeyHandlers) Rotate(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[ApiKeyRotateRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("expires_in").Nullable().Number().Min(5 * 60).Max(MaxExpiresIn)
		v.Field("permissions").Optional().Object()
		v.Field("all_permissions").Optional().Boolean()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	result, err := h.plugin.Rotate(r.Context(), session.User, limen.GetParam(r, "id"), body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, result)
}
