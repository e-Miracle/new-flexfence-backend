package mysql

import (
	"fmt"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
	"gorm.io/gorm"
)

func (s *Store) UpdateEvent(
	eventID, organizationID string,
	title, description string,
	startAt, endAt time.Time,
	geofenceGpsTolerance string,
) (domain.Event, error) {
	event, found, err := s.GetEventForOrganization(eventID, organizationID)
	if err != nil {
		return domain.Event{}, err
	}
	if !found {
		return domain.Event{}, store.ErrEventNotFound
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Event{}, fmt.Errorf("%w: title is required", store.ErrInvalidSchedule)
	}
	if err := domain.ValidateEventSchedule(startAt, endAt); err != nil {
		return domain.Event{}, fmt.Errorf("%w: %v", store.ErrInvalidSchedule, err)
	}
	startAt = startAt.UTC()
	endAt = endAt.UTC()
	now := time.Now().UTC()
	if domain.EventIsLive(event, now) {
		if !startAt.Equal(event.StartAt.UTC()) || !endAt.Equal(event.EndAt.UTC()) {
			return domain.Event{}, store.ErrEventLive
		}
	}
	updatedEvent := event
	updatedEvent.Title = title
	updatedEvent.Description = strings.TrimSpace(description)
	updatedEvent.StartAt = startAt
	updatedEvent.EndAt = endAt
	tolerance := domain.NormalizeGeofenceGpsTolerance(geofenceGpsTolerance)
	if strings.TrimSpace(geofenceGpsTolerance) == "" {
		tolerance = domain.NormalizeGeofenceGpsTolerance(event.GeofenceGpsTolerance)
	}
	updatedEvent.GeofenceGpsTolerance = tolerance
	fences, err := s.ListFencesByEvent(eventID)
	if err != nil {
		return domain.Event{}, err
	}
	for _, fence := range fences {
		fs := fence.StartAt
		fe := fence.EndAt
		if _, _, err := domain.ResolveFenceSchedule(updatedEvent, &fs, &fe); err != nil {
			return domain.Event{}, fmt.Errorf(
				"%w: fence %q does not fit the updated schedule (%v)",
				store.ErrInvalidSchedule,
				fence.Name,
				err,
			)
		}
	}
	res := s.db.Model(&EventModel{}).
		Where("id = ? AND organization_id = ?", eventID, organizationID).
		Updates(map[string]any{
			"title":                  title,
			"description":            strings.TrimSpace(description),
			"start_at":               startAt,
			"end_at":                 endAt,
			"geofence_gps_tolerance": tolerance,
		})
	if res.Error != nil {
		return domain.Event{}, res.Error
	}
	if res.RowsAffected == 0 {
		return domain.Event{}, store.ErrEventNotFound
	}
	updated, found, err := s.GetEventForOrganization(eventID, organizationID)
	if err != nil || !found {
		return domain.Event{}, store.ErrEventNotFound
	}
	return updated, nil
}

func (s *Store) DeleteEvent(eventID, organizationID string) error {
	event, found, err := s.GetEventForOrganization(eventID, organizationID)
	if err != nil {
		return err
	}
	if !found {
		return store.ErrEventNotFound
	}
	if domain.EventIsLive(event, time.Now().UTC()) {
		return store.ErrEventLive
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("event_id = ?", eventID).Delete(&GeofenceAlertModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("event_id = ?", eventID).Delete(&FenceCaptureSessionModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("event_id = ?", eventID).Delete(&UserConsentModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("event_id = ?", eventID).Delete(&ConsentTemplateModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("event_id = ?", eventID).Delete(&AttendanceModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("event_id = ?", eventID).Delete(&UserActivitySessionModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("event_id = ?", eventID).Delete(&EventJoinModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("event_id = ?", eventID).Delete(&FenceModel{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ? AND organization_id = ?", eventID, organizationID).Delete(&EventModel{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return store.ErrEventNotFound
		}
		return nil
	})
}
