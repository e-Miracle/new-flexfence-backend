package domain

import (
	"fmt"
	"regexp"
	"strings"
)

var consentKeySanitizer = regexp.MustCompile(`[^a-z0-9_]+`)

var allowedConsentValueTypes = map[string]struct{}{
	ConsentValueText:    {},
	ConsentValueEmail:   {},
	ConsentValuePhone:   {},
	ConsentValueNumber:  {},
	ConsentValueDate:    {},
	ConsentValueBoolean: {},
}

var presetConsentFields = map[string]ConsentField{
	"email":      {Key: "email", Label: "Email", Required: true, ValueType: ConsentValueEmail},
	"first_name": {Key: "first_name", Label: "First name", Required: true, ValueType: ConsentValueText},
	"last_name":  {Key: "last_name", Label: "Last name", Required: false, ValueType: ConsentValueText},
	"phone":      {Key: "phone", Label: "Phone number", Required: false, ValueType: ConsentValuePhone},
}

// NormalizeConsentFields validates and normalizes preset + custom fields for an event template.
func NormalizeConsentFields(fields []ConsentField) ([]ConsentField, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("at least one consent field is required")
	}
	seen := make(map[string]struct{}, len(fields))
	out := make([]ConsentField, 0, len(fields))
	for _, f := range fields {
		normalized, err := normalizeOneConsentField(f)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[normalized.Key]; dup {
			return nil, fmt.Errorf("duplicate field key: %s", normalized.Key)
		}
		seen[normalized.Key] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}

func normalizeOneConsentField(f ConsentField) (ConsentField, error) {
	label := strings.TrimSpace(f.Label)
	if f.IsCustom {
		if label == "" {
			return ConsentField{}, fmt.Errorf("custom field label is required")
		}
		valueType := strings.TrimSpace(strings.ToLower(f.ValueType))
		if valueType == "" {
			valueType = ConsentValueText
		}
		if _, ok := allowedConsentValueTypes[valueType]; !ok {
			return ConsentField{}, fmt.Errorf("invalid value_type for custom field %q", label)
		}
		key := strings.TrimSpace(strings.ToLower(f.Key))
		if key == "" {
			key = SlugConsentFieldKey(label)
		} else {
			key = consentKeySanitizer.ReplaceAllString(key, "_")
			key = strings.Trim(key, "_")
			if !strings.HasPrefix(key, "custom_") {
				key = "custom_" + key
			}
		}
		if key == "custom_" || key == "" {
			return ConsentField{}, fmt.Errorf("could not derive key for custom field %q", label)
		}
		return ConsentField{
			Key:       key,
			Label:     label,
			Required:  f.Required,
			IsCustom:  true,
			ValueType: valueType,
		}, nil
	}

	key := strings.TrimSpace(strings.ToLower(f.Key))
	preset, ok := presetConsentFields[key]
	if !ok {
		return ConsentField{}, fmt.Errorf("unknown preset field: %s", key)
	}
	required := f.Required
	if key == "email" || key == "first_name" {
		required = true
	}
	return ConsentField{
		Key:       preset.Key,
		Label:     preset.Label,
		Required:  required,
		IsCustom:  false,
		ValueType: preset.ValueType,
	}, nil
}

// SlugConsentFieldKey builds a stable custom_* key from a human label.
func SlugConsentFieldKey(label string) string {
	slug := strings.ToLower(strings.TrimSpace(label))
	slug = consentKeySanitizer.ReplaceAllString(strings.ReplaceAll(slug, " ", "_"), "_")
	slug = strings.Trim(slug, "_")
	if slug == "" {
		slug = "field"
	}
	if len(slug) > 48 {
		slug = slug[:48]
	}
	return "custom_" + slug
}
