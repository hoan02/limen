package organization

import (
	"context"
	"net/http"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type activeOrganizationContextKey struct{}

type SessionActiveOrganization struct {
	Session      *limen.ValidatedSession
	Organization *Organization
}

func WithActiveOrganization(ctx context.Context, session *limen.ValidatedSession, organization *Organization) context.Context {
	return context.WithValue(ctx, activeOrganizationContextKey{}, &SessionActiveOrganization{Session: session, Organization: organization})
}

func GetActiveOrganizationFromCtx(ctx context.Context) (*SessionActiveOrganization, error) {
	activeOrganization := ctx.Value(activeOrganizationContextKey{})
	if activeOrganization == nil {
		return nil, ErrNoActiveOrganization
	}
	return activeOrganization.(*SessionActiveOrganization), nil
}

func CurrentOrganizationIDFromCtx(ctx context.Context) (*limen.ValidatedSession, any, error) {
	session, err := limen.GetCurrentSessionFromCtx(ctx)
	if err != nil {
		return nil, nil, err
	}
	if session.Session.Metadata[MetadataActiveOrganizationID] == nil {
		return nil, nil, ErrNoActiveOrganization
	}
	return session, session.Session.Metadata[MetadataActiveOrganizationID], nil
}

func (o *organizationPlugin) HasPermissionMiddleware(permissions access.Permissions) limen.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, organizationID, err := CurrentOrganizationIDFromCtx(r.Context())
			if err != nil {
				o.responder.Error(w, r, err)
				return
			}

			if err := o.HasPermission(r.Context(), session.User, organizationID, permissions); err != nil {
				o.responder.Error(w, r, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (o *organizationPlugin) CanAccessOrganizationMiddleware() limen.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, organizationID, err := CurrentOrganizationIDFromCtx(r.Context())
			if err != nil {
				o.responder.Error(w, r, err)
				return
			}

			if err := o.CheckMemberExistsInOrganization(r.Context(), organizationID, session.User.ID); err != nil {
				o.responder.Error(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
