package mysql

import (
	"context"
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
)

func (s *Store) DeleteFence(eventID, fenceID string) error {
	event, ok, err := s.GetEvent(eventID)
	if err != nil {
		return err
	}
	if !ok {
		return store.ErrEventNotFound
	}
	if domain.EventIsLive(event, time.Now().UTC()) {
		return store.ErrEventLive
	}
	if _, ok, err := s.GetFenceByEvent(eventID, fenceID); err != nil {
		return err
	} else if !ok {
		return store.ErrFenceNotFound
	}
	res := s.db.Where("id = ? AND event_id = ?", fenceID, eventID).Delete(&FenceModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return store.ErrFenceNotFound
	}
	return nil
}

func (s *Store) clockOutOpenSessionsForEvent(eventID string) (int64, error) {
	now := time.Now().UTC()
	res := s.db.Model(&UserActivitySessionModel{}).
		Where("event_id = ? AND clock_out_at IS NULL", eventID).
		Update("clock_out_at", now)
	return res.RowsAffected, res.Error
}

func (s *Store) createGeofenceAlertsForEventJoins(event domain.Event, alertType, message string, fenceID, fenceName string) error {
	joins, err := s.ListJoinsByEvent(event.ID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, join := range joins {
		alert := GeofenceAlertModel{
			ID:         newGeofenceAlertID(now),
			UserID:     join.UserID,
			EventID:    event.ID,
			AlertType:  alertType,
			EventTitle: event.Title,
			FenceID:    fenceID,
			FenceName:  fenceName,
			Message:    message,
			CreatedAt:  now,
		}
		if err := s.db.Create(&alert).Error; err != nil {
			return err
		}
		if s.notifier != nil {
			mapped := mapGeofenceAlertModel(alert)
			go s.notifier.DispatchGeofenceAlert(context.Background(), join.UserID, mapped)
		}
		now = now.Add(time.Nanosecond)
	}
	return nil
}

func (s *Store) ProcessPendingEventGoLive(now time.Time) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var events []EventModel
	err := s.db.Where("go_live_processed_at IS NULL AND start_at <= ? AND end_at >= ?", now.UTC(), now.UTC()).
		Find(&events).Error
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, row := range events {
		event := mapEventModel(row)
		if !domain.EventIsLive(event, now) {
			continue
		}
		if _, err := s.clockOutOpenSessionsForEvent(event.ID); err != nil {
			return processed, err
		}
		msg := event.Title + " is now live. Any open fence sessions were closed."
		if err := s.createGeofenceAlertsForEventJoins(event, "event_live", msg, "", ""); err != nil {
			return processed, err
		}
		stamp := now.UTC()
		if err := s.db.Model(&EventModel{}).Where("id = ?", event.ID).Update("go_live_processed_at", stamp).Error; err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *Store) ConsumeUserGeofenceAlerts(userID string) ([]domain.GeofenceAlert, error) {
	var rows []GeofenceAlertModel
	err := s.db.Where("user_id = ? AND delivered_at IS NULL", userID).
		Order("created_at asc").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []domain.GeofenceAlert{}, nil
	}
	now := time.Now().UTC()
	ids := make([]string, 0, len(rows))
	out := make([]domain.GeofenceAlert, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
		out = append(out, mapGeofenceAlertModel(row))
	}
	if err := s.db.Model(&GeofenceAlertModel{}).
		Where("id IN ?", ids).
		Update("delivered_at", now).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func mapGeofenceAlertModel(row GeofenceAlertModel) domain.GeofenceAlert {
	return domain.GeofenceAlert{
		ID:         row.ID,
		Type:       row.AlertType,
		EventID:    row.EventID,
		EventTitle: row.EventTitle,
		FenceID:    row.FenceID,
		FenceName:  row.FenceName,
		Message:    row.Message,
		CreatedAt:  row.CreatedAt.UTC(),
	}
}

func newGeofenceAlertID(at time.Time) string {
	return "alert_" + at.UTC().Format("20060102150405.000000000")
}
