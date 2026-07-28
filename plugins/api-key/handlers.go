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

func (h *apiKeyHandlers) validateAPIKeyIDParam(v *limen.Validator) {
	v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
		return limen.ValidateClientIDValue(h.plugin.core, h.plugin.apiKeySchema, value)
	})
}

func newApiKeyCreateResponse(result *ApiKeyCreateResult, plugin *apiKeyPlugin) map[string]any {
	payload := plugin.core.SerializeModel(plugin.apiKeySchema, result.ApiKey)
	payload["key"] = result.Key
	return payload
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

	result, err := h.plugin.Create(r.Context(), session.User, body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	h.responder.JSON(w, r, http.StatusCreated, newApiKeyCreateResponse(result, h.plugin))
}

func (h *apiKeyHandlers) Get(w http.ResponseWriter, r *http.Request) {
	if limen.ValidateRequest(w, r, h.responder, h.validateAPIKeyIDParam) == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	apiKey, err := h.plugin.Get(r.Context(), session.User, limen.GetParam(r, "id"))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.apiKeySchema, apiKey))
}

func (h *apiKeyHandlers) List(w http.ResponseWriter, r *http.Request) {
	filter := limen.BindAndValidate[ApiKeyListFilter](w, r, h.responder, func(v *limen.Validator) {
		v.Field("profile").Optional(defaultProfile().ID).String().In(h.plugin.ProfileIDs())
		v.Field("status").Optional().String().In([]string{string(APIKeyStatusEnabled), string(APIKeyStatusDisabled)})
	})

	if filter == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	page, err := h.plugin.List(r.Context(), session.User, filter.ProfileID, filter, limen.ParsePagination(r))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, &limen.Page[map[string]any]{
		Items:      limen.SerializeModels(h.plugin.core, h.plugin.apiKeySchema, page.Items),
		Total:      page.Total,
		Page:       page.Page,
		PerPage:    page.PerPage,
		TotalPages: page.TotalPages,
	})
}

func (h *apiKeyHandlers) Update(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[ApiKeyUpdateRequest](w, r, h.responder, func(v *limen.Validator) {
		h.validateAPIKeyIDParam(v)
		v.Field("name").Optional().String()
		v.Field("permissions").Optional().Object()
		v.Field("all_permissions").Optional().Boolean()
		v.Field("enabled").Optional().Boolean()
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

	result, err := h.plugin.Update(r.Context(), session.User, limen.GetParam(r, "id"), body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.apiKeySchema, result))
}

func (h *apiKeyHandlers) Revoke(w http.ResponseWriter, r *http.Request) {
	if limen.ValidateRequest(w, r, h.responder, h.validateAPIKeyIDParam) == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
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
		h.validateAPIKeyIDParam(v)
		v.Field("expires_in").Nullable().Number().Min(5 * 60).Max(MaxExpiresIn)
		v.Field("permissions").Optional().Object()
		v.Field("all_permissions").Optional().Boolean()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	result, err := h.plugin.Rotate(r.Context(), session.User, limen.GetParam(r, "id"), body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	h.responder.JSON(w, r, http.StatusOK, newApiKeyCreateResponse(result, h.plugin))
}
