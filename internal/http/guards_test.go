package http

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/flexfence/flexfence-backend/internal/domain"
)

func mustURL(path string) *url.URL {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return u
}

func TestCanBusinessWrite(t *testing.T) {
	if !canBusinessWrite(domain.BusinessRoleOwner) {
		t.Fatal("owner should write")
	}
	if !canBusinessWrite(domain.BusinessRoleAdmin) {
		t.Fatal("admin should write")
	}
	if canBusinessWrite(domain.BusinessRoleViewer) {
		t.Fatal("viewer should not write")
	}
}

func TestClassifyEventRoute(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   eventRouteKind
	}{
		{http.MethodGet, "/v1/events/evt_1", eventRouteGet},
		{http.MethodPost, "/v1/events/evt_1/fences", eventRouteFence},
		{http.MethodPost, "/v1/events/evt_1/join-by-qr", eventRouteJoinQR},
		{http.MethodPost, "/v1/events/evt_1/attendance/mark-present", eventRouteMarkPresent},
		{http.MethodGet, "/v1/events/evt_1/fences", eventRouteFencesList},
		{http.MethodGet, "/v1/events/evt_1/attendance", eventRouteAttendanceList},
		{http.MethodGet, "/v1/events/evt_1/consent-template", eventRouteConsent},
		{http.MethodGet, "/v1/events/evt_1/share", eventRouteShare},
		{http.MethodPost, "/v1/events/evt_1/share/regenerate", eventRouteShareRegenerate},
		{http.MethodGet, "/v1/events/evt_1/clock-in-share", eventRouteClockInShare},
		{http.MethodPatch, "/v1/events/evt_1/clock-in-settings", eventRouteClockInSettings},
		{http.MethodPost, "/v1/events/evt_1/clock-in-share/regenerate", eventRouteClockInShareRegenerate},
		{http.MethodGet, "/v1/events/evt_1/analytics", eventRouteEventAnalytics},
		{http.MethodGet, "/v1/events/evt_1/fences/fence_1/analytics", eventRouteFenceAnalytics},
	}

	for _, tc := range cases {
		r := &http.Request{Method: tc.method, URL: mustURL(tc.path)}
		if got := classifyEventRoute(r); got != tc.want {
			t.Fatalf("%s %s: got %v want %v", tc.method, tc.path, got, tc.want)
		}
	}
}
