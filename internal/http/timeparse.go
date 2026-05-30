package http

import (
	"strings"
	"time"
)

func parseRFC3339Time(raw, field string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05", raw)
	}
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func parseOptionalRFC3339Time(raw, field string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	t, err := parseRFC3339Time(raw, field)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
