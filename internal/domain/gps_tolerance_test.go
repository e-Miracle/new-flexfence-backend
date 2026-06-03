package domain

import "testing"

func TestGeofenceGpsTolerancePresets(t *testing.T) {
	cases := map[string]struct {
		maxAccuracyM float64
		fenceBufferM float64
	}{
		GeofenceGpsToleranceNone:      {10, 0},
		GeofenceGpsToleranceStrict:    {10, 10},
		GeofenceGpsToleranceDefault:   {20, 20},
		GeofenceGpsToleranceForgiving: {35, 35},
		"":                            {20, 20},
		"unknown":                     {20, 20},
	}
	for input, want := range cases {
		if got := GeofenceGpsToleranceMaxAccuracyM(input); got != want.maxAccuracyM {
			t.Fatalf("MaxAccuracyM(%q) = %v, want %v", input, got, want.maxAccuracyM)
		}
		if got := GeofenceGpsToleranceFenceBufferM(input); got != want.fenceBufferM {
			t.Fatalf("FenceBufferM(%q) = %v, want %v", input, got, want.fenceBufferM)
		}
	}
}
