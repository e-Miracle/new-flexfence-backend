package domain

import "testing"

func TestPointInCircleFenceWithAccuracyAllowsGpsBuffer(t *testing.T) {
	fence := Fence{
		ShapeType: "circle",
		CenterLat: 6.5244,
		CenterLng: 3.3792,
		RadiusM:   50,
	}
	// ~55 m north of center — outside a 50 m fence but within 50 m + 10 m accuracy.
	lat := 6.5249
	lng := 3.3792
	if PointInFence(fence, lat, lng) {
		t.Fatal("expected raw point to be outside fence")
	}
	if !PointInFenceWithAccuracy(fence, lat, lng, 10, 10) {
		t.Fatal("expected accuracy buffer to include near-boundary fix")
	}
}

func TestPointInCircleFenceWithAccuracyRejectsLargeMiss(t *testing.T) {
	fence := Fence{
		ShapeType: "circle",
		CenterLat: 6.5244,
		CenterLng: 3.3792,
		RadiusM:   50,
	}
	lat := 6.5260
	lng := 3.3792
	if PointInFenceWithAccuracy(fence, lat, lng, 10, 10) {
		t.Fatal("expected far outside point to remain rejected")
	}
}
