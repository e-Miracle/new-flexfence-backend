package http

import (
	"net/http"
	"strings"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/domain"
	mysqlstore "github.com/flexfence/flexfence-backend/internal/store/mysql"
)

// Guards centralizes authentication and authorization middleware.
type Guards struct {
	tokens   *auth.TokenService
	identity *mysqlstore.IdentityStore
}

func NewGuards(tokens *auth.TokenService, identity *mysqlstore.IdentityStore) *Guards {
	return &Guards{tokens: tokens, identity: identity}
}

// Public routes do not require authentication (login, health).
func (g *Guards) Public(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// BusinessRead allows any active dashboard role (owner, admin, viewer).
func (g *Guards) BusinessRead(next http.Handler) http.Handler {
	return g.businessAuth(next, false)
}

// BusinessWrite allows owner and admin only (mutating dashboard actions).
func (g *Guards) BusinessWrite(next http.Handler) http.Handler {
	return g.businessAuth(next, true)
}

// User requires a valid mobile end-user JWT and active account.
func (g *Guards) User(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := g.parseBearer(w, r)
		if !ok {
			return
		}
		if claims.Type != auth.TokenTypeUser {
			writeAPIError(w, http.StatusForbidden, "forbidden", "End-user token required")
			return
		}
		if claims.Subject == "" {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Invalid user token claims")
			return
		}

		user, found, err := g.identity.GetUserByID(claims.Subject)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !found {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User account not found")
			return
		}
		if user.Status != domain.StatusActive {
			writeAPIError(w, http.StatusForbidden, "account_disabled", "User account is disabled")
			return
		}

		ctx := withUserAuth(r.Context(), UserAuth{
			UserID: user.ID,
			Email:  user.Email,
			Role:   "attendee",
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalUser enriches request context when a valid end-user Bearer token exists.
// Missing auth is allowed; invalid provided auth is rejected.
func (g *Guards) OptionalUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if header == "" {
			next.ServeHTTP(w, r)
			return
		}
		claims, ok := g.parseBearer(w, r)
		if !ok {
			return
		}
		if claims.Type != auth.TokenTypeUser {
			writeAPIError(w, http.StatusForbidden, "forbidden", "End-user token required")
			return
		}
		if claims.Subject == "" {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Invalid user token claims")
			return
		}
		user, found, err := g.identity.GetUserByID(claims.Subject)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !found {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "User account not found")
			return
		}
		if user.Status != domain.StatusActive {
			writeAPIError(w, http.StatusForbidden, "account_disabled", "User account is disabled")
			return
		}
		ctx := withUserAuth(r.Context(), UserAuth{
			UserID: user.ID,
			Email:  user.Email,
			Role:   "attendee",
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AllowMethods rejects requests whose HTTP method is not in the allowed list.
func (g *Guards) AllowMethods(methods ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(methods))
	for _, m := range methods {
		allowed[strings.ToUpper(m)] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := allowed[r.Method]; !ok {
				writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireEventTenant ensures the path event belongs to the authenticated business organization.
func (g *Guards) RequireEventTenant(store EventTenantStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			biz, ok := businessAuthFromContext(r.Context())
			if !ok {
				writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Business auth required")
				return
			}
			eventID := eventIDFromPath(r.URL.Path)
			if eventID == "" {
				writeAPIError(w, http.StatusBadRequest, "invalid_path", "Event id missing from path")
				return
			}
			_, found, err := store.GetEventForOrganization(eventID, biz.OrganizationID)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
				return
			}
			if !found {
				writeAPIError(w, http.StatusNotFound, "event_not_found", "Event was not found")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Chain composes middleware outer-to-inner (first listed runs first).
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		h := final
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}

func (g *Guards) businessAuth(next http.Handler, writeRequired bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := g.parseBearer(w, r)
		if !ok {
			return
		}
		if claims.Type != auth.TokenTypeBusiness {
			writeAPIError(w, http.StatusForbidden, "forbidden", "Business token required")
			return
		}
		if claims.Subject == "" || claims.OrganizationID == "" || claims.Role == "" {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Invalid business token claims")
			return
		}

		user, found, err := g.identity.GetBusinessUserByID(claims.Subject)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			return
		}
		if !found {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Business account not found")
			return
		}
		if user.Status != domain.StatusActive {
			writeAPIError(w, http.StatusForbidden, "account_disabled", "Business account is disabled")
			return
		}
		if user.OrganizationID != claims.OrganizationID {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Token organization mismatch")
			return
		}
		if user.Role != claims.Role {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Token role mismatch")
			return
		}

		if writeRequired && !canBusinessWrite(user.Role) {
			writeAPIError(w, http.StatusForbidden, "insufficient_role", "Owner or admin role required")
			return
		}
		if isWriteMethod(r.Method) && !canBusinessWrite(user.Role) {
			writeAPIError(w, http.StatusForbidden, "insufficient_role", "Owner or admin role required for this action")
			return
		}

		ctx := withBusinessAuth(r.Context(), BusinessAuth{
			UserID:         user.ID,
			OrganizationID: user.OrganizationID,
			Role:           user.Role,
			Email:          user.Email,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (g *Guards) parseBearer(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Missing Bearer token")
		return nil, false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	claims, err := g.tokens.Parse(raw)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
		return nil, false
	}
	return claims, true
}

func canBusinessWrite(role string) bool {
	return role == domain.BusinessRoleOwner || role == domain.BusinessRoleAdmin
}

func isWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func eventIDFromPath(path string) string {
	path = strings.TrimPrefix(path, "/v1/events/")
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	return strings.Split(path, "/")[0]
}

func fenceIDFromPath(path string) string {
	path = strings.TrimPrefix(path, "/v1/events/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) >= 3 && parts[1] == "fences" {
		return parts[2]
	}
	return ""
}

// EventTenantStore is the subset of store used for tenant checks.
type EventTenantStore interface {
	GetEventForOrganization(eventID, organizationID string) (domain.Event, bool, error)
}
