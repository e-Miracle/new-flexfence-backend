package domain

import (
	"testing"
	"time"
)

func TestComputeEventAnalytics_fenceBreakdown(t *testing.T) {
	now := time.Now().UTC()
	fences := []Fence{{ID: "f1", Name: "Hall", ShapeType: "circle", CenterLat: 1, CenterLng: 1, RadiusM: 100}}
	attendance := []AttendanceWithUser{
		{UserID: "u1", Source: "geofence_prompt", MarkedAt: now, Lat: 1, Lng: 1, FenceID: "f1"},
		{UserID: "u2", Source: "qr_scan", MarkedAt: now},
	}
	out := ComputeEventAnalytics(AnalyticsInput{
		Event:      Event{ID: "e1"},
		Fences:     fences,
		Joins:      []EventJoin{{UserID: "u1", JoinSource: "qr"}},
		Attendance: attendance,
	})
	if out.TotalAttendance != 2 || out.FenceSummaries[0].TotalAttendance != 1 {
		t.Fatalf("unexpected analytics: %+v", out)
	}
}
