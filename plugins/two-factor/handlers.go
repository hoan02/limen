package twofactor

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

type twoFactorHandlers struct {
	plugin    *twoFactorPlugin
	responder *limen.Responder
	httpCore  *limen.LimenHTTPCore
}

func newTwoFactorHandlers(plugin *twoFactorPlugin, responder *limen.Responder, httpCore *limen.LimenHTTPCore) *twoFactorHandlers {
	return &twoFactorHandlers{
		plugin:    plugin,
		responder: responder,
		httpCore:  httpCore,
	}
}

func (a *twoFactorHandlers) InitiateTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, a.responder, func(v *limen.Validator) {
		v.Field("password").Required()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	user := a.plugin.userSchema.UserToUserWithTwoFactor(session.User)
	result, err := a.plugin.InitiateTwoFactorSetup(r.Context(), user, body["password"].(string))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.JSON(w, r, http.StatusOK, result)
}

func (a *twoFactorHandlers) FinalizeTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, a.responder, func(v *limen.Validator) {
		v.Field("code").Required()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	user := a.plugin.userSchema.UserToUserWithTwoFactor(session.User)
	err = a.plugin.FinalizeTwoFactorSetup(r.Context(), user, body["code"].(string))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	authResult, sessionResult, err := a.plugin.core.RotateSession(r, w, session, a.plugin.config.revokeOtherSessionsOnStateChange)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.SessionResponse(w, r, a.plugin.core, authResult, sessionResult)
}

// Disable disables 2FA for the current user
func (a *twoFactorHandlers) Disable(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, a.responder, func(v *limen.Validator) {
		v.Field("password").Required()
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	err = a.plugin.DisableTwoFactor(r.Context(), session.User.ID, body["password"].(string))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	authResult, sessionResult, err := a.plugin.core.RotateSession(r, w, session, a.plugin.config.revokeOtherSessionsOnStateChange)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.SessionResponse(w, r, a.plugin.core, authResult, sessionResult)
}

// VerifyLoginWithTwoFactor verifies the 2FA code and completes the login process
func (a *twoFactorHandlers) VerifyLoginWithTwoFactor(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, a.responder, func(v *limen.Validator) {
		v.Field("code").Required()
		v.Field("method").
			Optional(string(TwoFactorMethodTOTP)).
			In([]string{string(TwoFactorMethodOTP), string(TwoFactorMethodTOTP)})
	})

	if body == nil {
		return
	}

	authResult, sessionResult, err := a.plugin.VerifyLoginWithTwoFactor(r, w, body["code"].(string), TwoFactorMethod(body["method"].(string)))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.SessionResponse(w, r, a.plugin.core, authResult, sessionResult)
}
