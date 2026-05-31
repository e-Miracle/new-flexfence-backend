package domain

import "errors"

const MaxLocationAccuracyM = 20.0

var (
	ErrMockLocation         = errors.New("mock_location")
	ErrLocationAccuracyPoor = errors.New("location_accuracy_poor")
	ErrInvalidCoordinates   = errors.New("invalid_coordinates")
	ErrOutsideFence         = errors.New("outside_fence")
)

type LocationReport struct {
	Lat          float64
	Lng          float64
	AccuracyM    float64
	MockLocation bool
}

func ValidateLocationReport(report LocationReport) error {
	if report.MockLocation {
		return ErrMockLocation
	}
	if report.Lat == 0 && report.Lng == 0 {
		return ErrInvalidCoordinates
	}
	if report.AccuracyM > 0 && report.AccuracyM > MaxLocationAccuracyM {
		return ErrLocationAccuracyPoor
	}
	return nil
}

// ValidateStrictLocationReport requires a positive accuracy estimate within MaxLocationAccuracyM.
func ValidateStrictLocationReport(report LocationReport) error {
	if err := ValidateLocationReport(report); err != nil {
		return err
	}
	if report.AccuracyM <= 0 || report.AccuracyM > MaxLocationAccuracyM {
		return ErrLocationAccuracyPoor
	}
	return nil
}

func ValidateGeofenceClockIn(fence Fence, lat, lng float64) error {
	if !PointInFence(fence, lat, lng) {
		return ErrOutsideFence
	}
	return nil
}
