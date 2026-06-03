package domain

import "strings"

const (
	GeofenceGpsToleranceNone      = "none"
	GeofenceGpsToleranceStrict    = "strict"
	GeofenceGpsToleranceDefault   = "default"
	GeofenceGpsToleranceForgiving = "forgiving"
)

// NormalizeGeofenceGpsTolerance maps organizer input to a supported preset.
func NormalizeGeofenceGpsTolerance(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case GeofenceGpsToleranceNone:
		return GeofenceGpsToleranceNone
	case GeofenceGpsToleranceStrict:
		return GeofenceGpsToleranceStrict
	case GeofenceGpsToleranceForgiving:
		return GeofenceGpsToleranceForgiving
	default:
		return GeofenceGpsToleranceDefault
	}
}

// GeofenceGpsToleranceMaxAccuracyM is the maximum acceptable GPS accuracy for a fix.
func GeofenceGpsToleranceMaxAccuracyM(tolerance string) float64 {
	switch NormalizeGeofenceGpsTolerance(tolerance) {
	case GeofenceGpsToleranceNone, GeofenceGpsToleranceStrict:
		return 10
	case GeofenceGpsToleranceForgiving:
		return 35
	default:
		return 20
	}
}

// GeofenceGpsToleranceFenceBufferM is how far beyond the fence edge GPS uncertainty may extend.
// Zero means the reported point must lie inside the fence with no allowance.
func GeofenceGpsToleranceFenceBufferM(tolerance string) float64 {
	switch NormalizeGeofenceGpsTolerance(tolerance) {
	case GeofenceGpsToleranceNone:
		return 0
	case GeofenceGpsToleranceStrict:
		return 10
	case GeofenceGpsToleranceForgiving:
		return 35
	default:
		return 20
	}
}

// GeofenceGpsToleranceMaxM returns the fence buffer meters for a preset.
func GeofenceGpsToleranceMaxM(tolerance string) float64 {
	return GeofenceGpsToleranceFenceBufferM(tolerance)
}
