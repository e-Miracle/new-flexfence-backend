package domain

import (
	"math"
	"time"
)

// DistanceMeters returns the haversine distance between two WGS84 points in meters.
func DistanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000
	rad := math.Pi / 180
	φ1 := lat1 * rad
	φ2 := lat2 * rad
	Δφ := (lat2 - lat1) * rad
	Δλ := (lng2 - lng1) * rad
	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * c
}

// PointInCircleFence reports whether a point lies inside a circular fence.
func PointInCircleFence(f Fence, lat, lng float64) bool {
	if f.ShapeType != "circle" || f.RadiusM <= 0 {
		return false
	}
	if lat == 0 && lng == 0 {
		return false
	}
	return DistanceMeters(f.CenterLat, f.CenterLng, lat, lng) <= f.RadiusM
}

// FenceActiveAt checks whether a timestamp falls within the fence schedule.
func FenceActiveAt(f Fence, at time.Time) bool {
	if at.IsZero() {
		return true
	}
	if !f.StartAt.IsZero() && at.Before(f.StartAt) {
		return false
	}
	if !f.EndAt.IsZero() && at.After(f.EndAt) {
		return false
	}
	return true
}

// PointInFence reports whether a point lies inside a fence shape.
func PointInFence(f Fence, lat, lng float64) bool {
	switch f.ShapeType {
	case "polygon":
		return PointInPolygonFence(f, lat, lng)
	default:
		return PointInCircleFence(f, lat, lng)
	}
}

// ResolveFenceForPoint picks the first fence containing the point during markedAt.
func ResolveFenceForPoint(fences []Fence, lat, lng float64, markedAt time.Time) string {
	for _, f := range fences {
		if !FenceActiveAt(f, markedAt) {
			continue
		}
		if PointInFence(f, lat, lng) {
			return f.ID
		}
	}
	return ""
}

// AttributeAttendanceFence returns fence_id for analytics (stored id or inferred).
func AttributeAttendanceFence(fenceID string, fences []Fence, lat, lng float64, markedAt time.Time) string {
	if fenceID != "" {
		return fenceID
	}
	return ResolveFenceForPoint(fences, lat, lng, markedAt)
}
