package organization

import (
	"context"
	"errors"
	"net/http"

	"github.com/thecodearcher/limen"
)

type activeOrganizationContextKey struct{}

type SessionActiveOrganization struct {
	Session      *limen.ValidatedSession
	Organization *Organization
}

func WithActiveOrganization(ctx context.Context, session *limen.ValidatedSession, organization *Organization) context.Context {
	return context.WithValue(ctx, activeOrganizationContextKey{}, &SessionActiveOrganization{Session: session, Organization: organization})
}

func GetActiveOrganizationSessionFromCtx(ctx context.Context) (*SessionActiveOrganization, error) {
	activeOrganization := ctx.Value(activeOrganizationContextKey{})
	if activeOrganization == nil {
		return nil, ErrNoActiveOrganization
	}
	return activeOrganization.(*SessionActiveOrganization), nil
}

func (o *organizationPlugin) GetActiveOrganizationIDFromCtx(ctx context.Context) (*limen.ValidatedSession, any, error) {
	session, err := limen.GetCurrentSessionFromCtx(ctx)
	if err != nil {
		return nil, nil, err
	}

	organizationID, err := o.GetActiveOrganizationID(ctx, session.Session)
	if err != nil {
		return nil, nil, err
	}
	if organizationID == nil {
		return nil, nil, ErrNoActiveOrganization
	}
	return session, organizationID, nil
}

func (o *organizationPlugin) RequireActiveOrganizationMiddleware() limen.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, organizationID, err := o.GetActiveOrganizationIDFromCtx(r.Context())
			if err != nil {
				o.responder.Error(w, r, err)
				return
			}

			organization, err := o.GetOrganization(r.Context(), organizationID)
			if err != nil {
				if errors.Is(err, limen.ErrRecordNotFound) {
					o.responder.Error(w, r, ErrNoActiveOrganization)
					return
				}
				o.responder.Error(w, r, err)
				return
			}
			r = r.WithContext(WithActiveOrganization(r.Context(), session, organization))
			next.ServeHTTP(w, r)
		})
	}
}
