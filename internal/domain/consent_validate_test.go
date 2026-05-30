package domain

import "testing"

func TestNormalizeConsentFields_Custom(t *testing.T) {
	fields := []ConsentField{
		{Key: "email", Label: "Email", Required: true},
		{Label: "Employee ID", Required: true, IsCustom: true, ValueType: ConsentValueText},
	}
	out, err := NormalizeConsentFields(fields)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(out))
	}
	if !out[1].IsCustom || out[1].Key != "custom_employee_id" {
		t.Fatalf("unexpected custom field: %+v", out[1])
	}
}
