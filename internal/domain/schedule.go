package domain

import (
	"fmt"
	"strings"
	"time"
)

// ValidateEventSchedule ensures event start/end are set and ordered.
func ValidateEventSchedule(startAt, endAt time.Time) error {
	if startAt.IsZero() || endAt.IsZero() {
		return fmt.Errorf("start_at and end_at are required")
	}
	if !endAt.After(startAt) {
		return fmt.Errorf("end_at must be after start_at")
	}
	return nil
}

// EventHasSchedule reports whether the event has both bounds for fence validation.
func EventHasSchedule(e Event) bool {
	return !e.StartAt.IsZero() && !e.EndAt.IsZero()
}

// EventIsLive reports whether the event is currently within its scheduled window.
func EventIsLive(e Event, at time.Time) bool {
	if !EventHasSchedule(e) || at.IsZero() {
		return false
	}
	at = at.UTC()
	start := e.StartAt.UTC()
	end := e.EndAt.UTC()
	return !at.Before(start) && !at.After(end)
}

// ResolveFenceSchedule applies optional fence times, inheriting from the event when omitted.
func ResolveFenceSchedule(event Event, fenceStart, fenceEnd *time.Time) (time.Time, time.Time, error) {
	var start, end time.Time
	if fenceStart != nil && !fenceStart.IsZero() {
		start = fenceStart.UTC()
	} else {
		start = event.StartAt.UTC()
	}
	if fenceEnd != nil && !fenceEnd.IsZero() {
		end = fenceEnd.UTC()
	} else {
		end = event.EndAt.UTC()
	}
	if start.IsZero() || end.IsZero() {
		return time.Time{}, time.Time{}, fmt.Errorf("fence schedule requires event start_at and end_at, or explicit fence times")
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("fence end_at must be after fence start_at")
	}
	if EventHasSchedule(event) {
		evStart := event.StartAt.UTC()
		evEnd := event.EndAt.UTC()
		if start.Before(evStart) {
			return time.Time{}, time.Time{}, fmt.Errorf("fence start_at cannot be before event start_at")
		}
		if end.After(evEnd) {
			return time.Time{}, time.Time{}, fmt.Errorf("fence end_at cannot be after event end_at")
		}
	}
	return start, end, nil
}

// GenerateFenceName builds a display name when the organizer leaves the field empty.
func GenerateFenceName(eventTitle string, startAt, endAt time.Time) string {
	title := strings.TrimSpace(eventTitle)
	if title == "" {
		title = "Event"
	}
	if startAt.IsZero() || endAt.IsZero() {
		return title + " area"
	}
	const layout = "Jan 2, 15:04"
	startStr := startAt.UTC().Format(layout)
	endStr := endAt.UTC().Format(layout)
	if startAt.UTC().Format("2006-01-02") == endAt.UTC().Format("2006-01-02") {
		endStr = endAt.UTC().Format("15:04")
	}
	return fmt.Sprintf("%s · %s–%s", title, startStr, endStr)
}
