package mysql

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
	"gorm.io/gorm"
)

func (s *Store) EventScanToClockInEnabled(eventID string) (bool, error) {
	var event EventModel
	if err := s.db.Select("scan_to_clock_in_enabled").Where("id = ?", eventID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, store.ErrEventNotFound
		}
		return false, err
	}
	return event.ScanToClockInEnabled, nil
}

func (s *Store) UserHasEventConsent(eventID, userID string) (bool, error) {
	var count int64
	err := s.db.Model(&UserConsentModel{}).
		Where("event_id = ? AND user_id = ?", eventID, userID).
		Count(&count).Error
	return count > 0, err
}

func (s *Store) GetUserEventConsent(eventID, userID string) (domain.UserConsent, bool, error) {
	var row UserConsentModel
	if err := s.db.Where("event_id = ? AND user_id = ?", eventID, userID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.UserConsent{}, false, nil
		}
		return domain.UserConsent{}, false, err
	}
	out, err := mapUserConsentModel(row)
	if err != nil {
		return domain.UserConsent{}, false, err
	}
	return out, true, nil
}

func (s *Store) SaveUserEventConsent(
	eventID, userID string,
	values map[string]string,
	tpl domain.ConsentTemplate,
) (domain.UserConsent, error) {
	if err := domain.ValidateConsentSubmission(tpl.RequiredFields, values); err != nil {
		return domain.UserConsent{}, fmt.Errorf("%w: %v", store.ErrInvalidConsent, err)
	}
	snapshot := make(map[string]string, len(values))
	for k, v := range values {
		snapshot[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return domain.UserConsent{}, err
	}
	now := time.Now().UTC()
	var existing UserConsentModel
	err = s.db.Where("event_id = ? AND user_id = ?", eventID, userID).First(&existing).Error
	if err == nil {
		existing.ConsentSnapshotJSON = string(raw)
		existing.AgreedAt = now
		if err := s.db.Save(&existing).Error; err != nil {
			return domain.UserConsent{}, err
		}
		return mapUserConsentModel(existing)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.UserConsent{}, err
	}
	row := UserConsentModel{
		ID:                  fmt.Sprintf("uconsent_%d", now.UnixNano()),
		EventID:             eventID,
		UserID:              userID,
		ConsentSnapshotJSON: string(raw),
		AgreedAt:            now,
		CreatedAt:           now,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return domain.UserConsent{}, err
	}
	return mapUserConsentModel(row)
}

func mapUserConsentModel(row UserConsentModel) (domain.UserConsent, error) {
	var snapshot map[string]string
	if err := json.Unmarshal([]byte(row.ConsentSnapshotJSON), &snapshot); err != nil {
		return domain.UserConsent{}, err
	}
	return domain.UserConsent{
		ID:              row.ID,
		EventID:         row.EventID,
		UserID:          row.UserID,
		ConsentSnapshot: snapshot,
		AgreedAt:        row.AgreedAt,
	}, nil
}
