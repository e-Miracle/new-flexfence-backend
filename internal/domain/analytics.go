package domain

import "time"

type TimelinePoint struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type FenceAnalyticsSummary struct {
	FenceID           string `json:"fence_id"`
	FenceName         string `json:"fence_name"`
	TotalAttendance   int    `json:"total_attendance"`
	UniqueAttendees   int    `json:"unique_attendees"`
	AttendanceInFence int    `json:"attendance_in_fence"`
}

type EventAnalytics struct {
	EventID             string                  `json:"event_id"`
	TotalJoins          int                     `json:"total_joins"`
	UniqueJoinedUsers   int                     `json:"unique_joined_users"`
	TotalAttendance     int                     `json:"total_attendance"`
	UniqueAttendees     int                     `json:"unique_attendees"`
	JoinsBySource       map[string]int          `json:"joins_by_source"`
	AttendanceBySource  map[string]int          `json:"attendance_by_source"`
	AttendanceTimeline  []TimelinePoint         `json:"attendance_timeline"`
	FenceSummaries      []FenceAnalyticsSummary `json:"fence_summaries"`
}

type FenceAnalytics struct {
	FenceID            string           `json:"fence_id"`
	FenceName          string           `json:"fence_name"`
	EventID            string           `json:"event_id"`
	TotalAttendance    int              `json:"total_attendance"`
	UniqueAttendees    int              `json:"unique_attendees"`
	AttendanceBySource map[string]int   `json:"attendance_by_source"`
	AttendanceTimeline []TimelinePoint  `json:"attendance_timeline"`
	UnattributedNearby int              `json:"unattributed_nearby"`
}

type AnalyticsInput struct {
	Event      Event
	Fences     []Fence
	Joins      []EventJoin
	Attendance []AttendanceWithUser
}

func ComputeEventAnalytics(in AnalyticsInput) EventAnalytics {
	joinUsers := make(map[string]struct{})
	joinsBySource := make(map[string]int)
	for _, j := range in.Joins {
		joinsBySource[j.JoinSource]++
		joinUsers[j.UserID] = struct{}{}
	}

	attendeeUsers := make(map[string]struct{})
	attBySource := make(map[string]int)
	fenceUsers := make(map[string]map[string]struct{})
	fenceCounts := make(map[string]int)

	var markedTimes []time.Time
	for _, a := range in.Attendance {
		attendeeUsers[a.UserID] = struct{}{}
		src := a.Source
		if src == "" {
			src = "unknown"
		}
		attBySource[src]++
		markedTimes = append(markedTimes, a.MarkedAt)

		fid := AttributeAttendanceFence(a.FenceID, in.Fences, a.Lat, a.Lng, a.MarkedAt)
		if fid != "" {
			fenceCounts[fid]++
			if fenceUsers[fid] == nil {
				fenceUsers[fid] = make(map[string]struct{})
			}
			fenceUsers[fid][a.UserID] = struct{}{}
		}
	}

	summaries := make([]FenceAnalyticsSummary, 0, len(in.Fences))
	for _, f := range in.Fences {
		unique := len(fenceUsers[f.ID])
		summaries = append(summaries, FenceAnalyticsSummary{
			FenceID:           f.ID,
			FenceName:         f.Name,
			TotalAttendance:   fenceCounts[f.ID],
			UniqueAttendees:   unique,
			AttendanceInFence: fenceCounts[f.ID],
		})
	}

	return EventAnalytics{
		EventID:            in.Event.ID,
		TotalJoins:         len(in.Joins),
		UniqueJoinedUsers:  len(joinUsers),
		TotalAttendance:    len(in.Attendance),
		UniqueAttendees:    len(attendeeUsers),
		JoinsBySource:      joinsBySource,
		AttendanceBySource: attBySource,
		AttendanceTimeline: buildTimeline(markedTimes, in.Event.StartAt, in.Event.EndAt),
		FenceSummaries:     summaries,
	}
}

func ComputeFenceAnalytics(fence Fence, event Event, fences []Fence, attendance []AttendanceWithUser) FenceAnalytics {
	attBySource := make(map[string]int)
	users := make(map[string]struct{})
	var markedTimes []time.Time
	unattributed := 0

	for _, a := range attendance {
		fid := AttributeAttendanceFence(a.FenceID, fences, a.Lat, a.Lng, a.MarkedAt)
		if fid != fence.ID {
			if fid == "" && PointInCircleFence(fence, a.Lat, a.Lng) {
				unattributed++
			}
			continue
		}
		users[a.UserID] = struct{}{}
		src := a.Source
		if src == "" {
			src = "unknown"
		}
		attBySource[src]++
		markedTimes = append(markedTimes, a.MarkedAt)
	}

	return FenceAnalytics{
		FenceID:            fence.ID,
		FenceName:          fence.Name,
		EventID:            fence.EventID,
		TotalAttendance:    len(markedTimes),
		UniqueAttendees:    len(users),
		AttendanceBySource: attBySource,
		AttendanceTimeline: buildTimeline(markedTimes, fence.StartAt, fence.EndAt),
		UnattributedNearby: unattributed,
	}
}

func buildTimeline(times []time.Time, windowStart, windowEnd time.Time) []TimelinePoint {
	if len(times) == 0 {
		return []TimelinePoint{}
	}
	useHourly := true
	if !windowStart.IsZero() && !windowEnd.IsZero() {
		if windowEnd.Sub(windowStart) > 48*time.Hour {
			useHourly = false
		}
	}
	buckets := make(map[string]int)
	for _, t := range times {
		var label string
		if useHourly {
			label = t.UTC().Format("Jan 2, 15:00")
		} else {
			label = t.UTC().Format("Jan 2, 2006")
		}
		buckets[label]++
	}
	// Preserve insertion order by re-walking times for stable sort
	seen := make(map[string]bool)
	out := make([]TimelinePoint, 0, len(buckets))
	for _, t := range times {
		var label string
		if useHourly {
			label = t.UTC().Format("Jan 2, 15:00")
		} else {
			label = t.UTC().Format("Jan 2, 2006")
		}
		if seen[label] {
			continue
		}
		seen[label] = true
		out = append(out, TimelinePoint{Label: label, Count: buckets[label]})
	}
	return out
}
