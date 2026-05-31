package mysql

import (
	"fmt"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
	"gorm.io/gorm"
)

const activityHistoryRetentionDays = 7

func (s *Store) ListUserActivityHistory(userID string, filter store.ActivityHistoryFilter) ([]domain.UserActivitySession, error) {
	query := s.db.Model(&UserActivitySessionModel{}).Where("user_id = ?", userID)

	now := time.Now().UTC()
	retentionCutoff := now.AddDate(0, 0, -activityHistoryRetentionDays)
	from, to := resolveActivityHistoryRange(filter, now, retentionCutoff)
	if from != nil {
		query = query.Where("clock_in_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("clock_in_at <= ?", *to)
	}

	var rows []UserActivitySessionModel
	if err := query.Order("clock_in_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserActivitySession, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapUserActivitySessionModel(row))
	}
	return out, nil
}

func (s *Store) RecordClockIn(userID, eventID, eventTitle, fenceID, fenceName, source string, lat, lng float64) (domain.UserActivitySession, error) {
	now := time.Now().UTC()
	if strings.TrimSpace(source) == "" {
		source = "manual"
	}
	verified := source == "geofence" || source == "qr_scan" || lat != 0 || lng != 0
	row := UserActivitySessionModel{
		ID:         fmt.Sprintf("act_%d", now.UnixNano()),
		UserID:     userID,
		EventID:    strings.TrimSpace(eventID),
		EventTitle: strings.TrimSpace(eventTitle),
		FenceID:    strings.TrimSpace(fenceID),
		FenceName:  strings.TrimSpace(fenceName),
		ClockInAt:  now,
		Verified:   verified,
		Source:     source,
		CreatedAt:  now,
	}
	if row.FenceName == "" {
		row.FenceName = "Unknown fence"
	}
	if err := s.db.Create(&row).Error; err != nil {
		return domain.UserActivitySession{}, err
	}
	return mapUserActivitySessionModel(row), nil
}

func (s *Store) RecordClockOut(userID, sessionID, eventID, fenceID string) (domain.UserActivitySession, error) {
	now := time.Now().UTC()
	var row UserActivitySessionModel
	query := s.db.Where("user_id = ? AND clock_out_at IS NULL", userID)
	switch {
	case strings.TrimSpace(sessionID) != "":
		query = query.Where("id = ?", strings.TrimSpace(sessionID))
	case strings.TrimSpace(eventID) != "" && strings.TrimSpace(fenceID) != "":
		query = query.Where("event_id = ? AND fence_id = ?", strings.TrimSpace(eventID), strings.TrimSpace(fenceID))
	default:
		return domain.UserActivitySession{}, store.ErrOpenSessionNotFound
	}
	if err := query.Order("clock_in_at desc").First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.UserActivitySession{}, store.ErrOpenSessionNotFound
		}
		return domain.UserActivitySession{}, err
	}
	row.ClockOutAt = &now
	if err := s.db.Save(&row).Error; err != nil {
		return domain.UserActivitySession{}, err
	}
	return mapUserActivitySessionModel(row), nil
}

func (s *Store) DeleteActivityHistoryOlderThan(cutoff time.Time) (int64, error) {
	res := s.db.Where("clock_in_at < ?", cutoff.UTC()).Delete(&UserActivitySessionModel{})
	return res.RowsAffected, res.Error
}

func resolveActivityHistoryRange(filter store.ActivityHistoryFilter, now, retentionCutoff time.Time) (*time.Time, *time.Time) {
	period := strings.TrimSpace(strings.ToLower(filter.Period))
	if period == "" {
		period = "date"
	}

	var from *time.Time
	var to *time.Time

	switch period {
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(weekday - 1))
		from = &start
		to = &now
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		from = &start
		to = &now
	case "custom":
		from = filter.From
		to = filter.To
		if to == nil {
			to = &now
		}
	default:
		from = &retentionCutoff
		to = &now
	}

	if from == nil || from.Before(retentionCutoff) {
		from = &retentionCutoff
	}
	if to == nil {
		to = &now
	}
	return from, to
}

func mapUserActivitySessionModel(row UserActivitySessionModel) domain.UserActivitySession {
	return domain.UserActivitySession{
		ID:         row.ID,
		UserID:     row.UserID,
		EventID:    row.EventID,
		EventTitle: row.EventTitle,
		FenceID:    row.FenceID,
		FenceName:  row.FenceName,
		ClockInAt:  row.ClockInAt.UTC(),
		ClockOutAt: row.ClockOutAt,
		Verified:   row.Verified,
		Source:     row.Source,
		CreatedAt:  row.CreatedAt.UTC(),
	}
}
