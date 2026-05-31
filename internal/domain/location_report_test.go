package domain

import "testing"

func TestValidateLocationReportRejectsMock(t *testing.T) {
	err := ValidateLocationReport(LocationReport{Lat: 1, Lng: 2, MockLocation: true})
	if err != ErrMockLocation {
		t.Fatalf("expected ErrMockLocation, got %v", err)
	}
}

func TestValidateLocationReportRejectsPoorAccuracy(t *testing.T) {
	err := ValidateLocationReport(LocationReport{Lat: 1, Lng: 2, AccuracyM: 25})
	if err != ErrLocationAccuracyPoor {
		t.Fatalf("expected ErrLocationAccuracyPoor, got %v", err)
	}
}

func TestValidateLocationReportAcceptsGoodFix(t *testing.T) {
	if err := ValidateLocationReport(LocationReport{Lat: 1, Lng: 2, AccuracyM: 12}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
