package domain

import (
	"testing"
	"time"
)

func TestResolveFenceSchedule_inheritsEvent(t *testing.T) {
	evStart := time.Date(2026, 3, 27, 9, 0, 0, 0, time.UTC)
	evEnd := time.Date(2026, 3, 27, 18, 0, 0, 0, time.UTC)
	event := Event{Title: "Conf", StartAt: evStart, EndAt: evEnd}

	start, end, err := ResolveFenceSchedule(event, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !start.Equal(evStart) || !end.Equal(evEnd) {
		t.Fatalf("expected event window, got %v – %v", start, end)
	}
}

func TestResolveFenceSchedule_rejectsBeforeEvent(t *testing.T) {
	evStart := time.Date(2026, 3, 27, 9, 0, 0, 0, time.UTC)
	evEnd := time.Date(2026, 3, 27, 18, 0, 0, 0, time.UTC)
	event := Event{StartAt: evStart, EndAt: evEnd}
	early := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	_, _, err := ResolveFenceSchedule(event, &early, nil)
	if err == nil {
		t.Fatal("expected error for fence starting before event")
	}
}

func TestGenerateFenceName(t *testing.T) {
	start := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	name := GenerateFenceName("Summit", start, end)
	if name == "" || name == "Summit area" {
		t.Fatalf("unexpected name: %q", name)
	}
}

func TestEventIsLive(t *testing.T) {
	start := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	event := Event{StartAt: start, EndAt: end}
	if EventIsLive(event, start.Add(-time.Minute)) {
		t.Fatal("expected event to be not live before start")
	}
	if !EventIsLive(event, start) {
		t.Fatal("expected event to be live at start")
	}
	if !EventIsLive(event, start.Add(time.Hour)) {
		t.Fatal("expected event to be live during window")
	}
	if EventIsLive(event, end.Add(time.Minute)) {
		t.Fatal("expected event to be not live after end")
	}
}
