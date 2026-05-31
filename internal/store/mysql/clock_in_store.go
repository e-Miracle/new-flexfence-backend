package mysql

import (
	"errors"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/store"
	"gorm.io/gorm"
)

const minClockInQRRotationMinutes = 5

func clockInQRIsExpired(issuedAt *time.Time, rotationMinutes int, now time.Time) bool {
	if rotationMinutes <= 0 || issuedAt == nil || issuedAt.IsZero() {
		return false
	}
	return !now.Before(issuedAt.Add(time.Duration(rotationMinutes) * time.Minute))
}

func (s *Store) UpdateEventClockInSettings(eventID string, enabled bool, rotationMinutes int) error {
	if rotationMinutes < 0 {
		return store.ErrInvalidSchedule
	}
	if rotationMinutes > 0 && rotationMinutes < minClockInQRRotationMinutes {
		return store.ErrInvalidSchedule
	}
	var event EventModel
	if err := s.db.Where("id = ?", eventID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return store.ErrEventNotFound
		}
		return err
	}
	updates := map[string]any{
		"scan_to_clock_in_enabled":     enabled,
		"clock_in_qr_rotation_minutes": rotationMinutes,
	}
	if !enabled {
		updates["clock_in_qr_token"] = ""
		updates["clock_in_qr_issued_at"] = nil
	}
	if err := s.db.Model(&event).Updates(updates).Error; err != nil {
		return err
	}
	if enabled && strings.TrimSpace(event.ClockInQRToken) == "" {
		_, _, err := s.EnsureEventClockInQRToken(eventID)
		return err
	}
	return nil
}

func (s *Store) EnsureEventClockInQRToken(eventID string) (string, time.Time, error) {
	var event EventModel
	if err := s.db.Where("id = ?", eventID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", time.Time{}, store.ErrEventNotFound
		}
		return "", time.Time{}, err
	}
	if !event.ScanToClockInEnabled {
		return "", time.Time{}, store.ErrClockInQRDisabled
	}
	now := time.Now().UTC()
	if strings.TrimSpace(event.ClockInQRToken) != "" &&
		!clockInQRIsExpired(event.ClockInQRIssuedAt, event.ClockInQRRotationMinutes, now) {
		issued := now
		if event.ClockInQRIssuedAt != nil {
			issued = event.ClockInQRIssuedAt.UTC()
		}
		return event.ClockInQRToken, issued, nil
	}
	return s.issueEventClockInQRToken(&event, now)
}

func (s *Store) RegenerateEventClockInQRToken(eventID string) (string, time.Time, error) {
	var event EventModel
	if err := s.db.Where("id = ?", eventID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", time.Time{}, store.ErrEventNotFound
		}
		return "", time.Time{}, err
	}
	if !event.ScanToClockInEnabled {
		return "", time.Time{}, store.ErrClockInQRDisabled
	}
	return s.issueEventClockInQRToken(&event, time.Now().UTC())
}

func (s *Store) issueEventClockInQRToken(event *EventModel, now time.Time) (string, time.Time, error) {
	token, err := auth.GenerateQRToken()
	if err != nil {
		return "", time.Time{}, err
	}
	issuedAt := now.UTC()
	if err := s.db.Model(event).Updates(map[string]any{
		"clock_in_qr_token":     token,
		"clock_in_qr_issued_at": issuedAt,
	}).Error; err != nil {
		return "", time.Time{}, err
	}
	return token, issuedAt, nil
}

func (s *Store) ValidateEventClockInQRToken(eventID, qrToken string) error {
	if strings.TrimSpace(qrToken) == "" {
		return store.ErrInvalidQRToken
	}
	var event EventModel
	if err := s.db.Where("id = ?", eventID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return store.ErrEventNotFound
		}
		return err
	}
	if !event.ScanToClockInEnabled {
		return store.ErrClockInQRDisabled
	}
	if qrToken != event.ClockInQRToken {
		return store.ErrInvalidQRToken
	}
	if clockInQRIsExpired(event.ClockInQRIssuedAt, event.ClockInQRRotationMinutes, time.Now().UTC()) {
		return store.ErrClockInQRExpired
	}
	return nil
}

func (s *Store) UserHasJoinedEvent(eventID, userID string) (bool, error) {
	var existing EventJoinModel
	err := s.db.Where("event_id = ? AND user_id = ?", eventID, userID).First(&existing).Error
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}
