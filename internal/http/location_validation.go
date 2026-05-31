package http

import (
	"errors"
	"net/http"

	"github.com/flexfence/flexfence-backend/internal/domain"
)

func writeLocationValidationErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrMockLocation):
		writeAPIError(w, http.StatusBadRequest, "mock_location", "Mock or spoofed GPS is not allowed")
	case errors.Is(err, domain.ErrLocationAccuracyPoor):
		writeAPIError(
			w,
			http.StatusBadRequest,
			"location_accuracy_poor",
			"GPS accuracy must be within 20 meters",
		)
	case errors.Is(err, domain.ErrInvalidCoordinates):
		writeAPIError(w, http.StatusBadRequest, "coordinates_required", "lat and lng are required")
	case errors.Is(err, domain.ErrOutsideFence):
		writeAPIError(w, http.StatusBadRequest, "outside_fence", "Reported location is outside the geofence")
	default:
		writeAPIError(w, http.StatusBadRequest, "invalid_location", "Location could not be validated")
	}
}

func validateLocationReport(lat, lng, accuracyM float64, mockLocation bool) error {
	return domain.ValidateLocationReport(domain.LocationReport{
		Lat:          lat,
		Lng:          lng,
		AccuracyM:    accuracyM,
		MockLocation: mockLocation,
	})
}

func validateStrictLocationReport(lat, lng, accuracyM float64, mockLocation bool) error {
	return domain.ValidateStrictLocationReport(domain.LocationReport{
		Lat:          lat,
		Lng:          lng,
		AccuracyM:    accuracyM,
		MockLocation: mockLocation,
	})
}
