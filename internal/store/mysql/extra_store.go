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

func (s *Store) ListFencesByEvent(eventID string) ([]domain.Fence, error) {
	var fences []FenceModel
	if err := s.db.Where("event_id = ?", eventID).Order("created_at desc").Find(&fences).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Fence, 0, len(fences))
	for _, f := range fences {
		out = append(out, mapFenceModel(f))
	}
	return out, nil
}

func (s *Store) ListAttendanceByEvent(eventID string) ([]domain.AttendanceWithUser, error) {
	type row struct {
		AttendanceModel
		UserEmail     string `gorm:"column:user_email"`
		UserFirstName string `gorm:"column:user_first_name"`
		UserLastName  string `gorm:"column:user_last_name"`
	}
	var rows []row
	err := s.db.Table("attendance_records AS a").
		Select("a.*, u.email AS user_email, u.first_name AS user_first_name, u.last_name AS user_last_name").
		Joins("LEFT JOIN users u ON u.id = a.user_id").
		Where("a.event_id = ?", eventID).
		Order("a.marked_at desc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.AttendanceWithUser, 0, len(rows))
	for _, r := range rows {
		name := strings.TrimSpace(r.UserFirstName + " " + r.UserLastName)
		out = append(out, domain.AttendanceWithUser{
			ID:        r.ID,
			EventID:   r.EventID,
			FenceID:   r.FenceID,
			UserID:    r.UserID,
			UserEmail: r.UserEmail,
			UserName:  name,
			Status:    r.Status,
			Source:    r.Source,
			MarkedAt:  r.MarkedAt,
			Lat:       r.Lat,
			Lng:       r.Lng,
			AccuracyM: r.AccuracyM,
		})
	}
	return out, nil
}

func (s *Store) GetConsentTemplate(eventID string) (domain.ConsentTemplate, bool, error) {
	var m ConsentTemplateModel
	err := s.db.Where("event_id = ?", eventID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ConsentTemplate{}, false, nil
	}
	if err != nil {
		return domain.ConsentTemplate{}, false, err
	}
	return mapConsentModel(m)
}

func (s *Store) SaveConsentTemplate(eventID string, tpl domain.ConsentTemplate) (domain.ConsentTemplate, error) {
	_, ok, err := s.GetEvent(eventID)
	if err != nil {
		return domain.ConsentTemplate{}, err
	}
	if !ok {
		return domain.ConsentTemplate{}, store.ErrEventNotFound
	}

	fieldsJSON, err := json.Marshal(tpl.RequiredFields)
	if err != nil {
		return domain.ConsentTemplate{}, err
	}
	now := time.Now().UTC()

	var existing ConsentTemplateModel
	err = s.db.Where("event_id = ?", eventID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		m := ConsentTemplateModel{
			ID:                 fmt.Sprintf("consent_%d", now.UnixNano()),
			EventID:            eventID,
			RequiredFieldsJSON: string(fieldsJSON),
			TrackEntryExit:     tpl.TrackEntryExit,
			TrackMovement:      tpl.TrackMovement,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := s.db.Create(&m).Error; err != nil {
			return domain.ConsentTemplate{}, err
		}
		mapped, _, mapErr := mapConsentModel(m)
		return mapped, mapErr
	}
	if err != nil {
		return domain.ConsentTemplate{}, err
	}

	existing.RequiredFieldsJSON = string(fieldsJSON)
	existing.TrackEntryExit = tpl.TrackEntryExit
	existing.TrackMovement = tpl.TrackMovement
	existing.UpdatedAt = now
	if err := s.db.Save(&existing).Error; err != nil {
		return domain.ConsentTemplate{}, err
	}
	mapped, _, mapErr := mapConsentModel(existing)
	return mapped, mapErr
}

func (s *Store) RecordOrganizationConsentFields(organizationID string, fields []domain.ConsentField) error {
	now := time.Now().UTC()
	for _, f := range fields {
		if !f.IsCustom {
			continue
		}
		var existing OrganizationConsentFieldModel
		err := s.db.Where("organization_id = ? AND field_key = ?", organizationID, f.Key).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			m := OrganizationConsentFieldModel{
				ID:             fmt.Sprintf("ocf_%d_%s", now.UnixNano(), f.Key),
				OrganizationID: organizationID,
				FieldKey:       f.Key,
				Label:          f.Label,
				ValueType:      f.ValueType,
				UseCount:       1,
				LastUsedAt:     now,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := s.db.Create(&m).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		existing.Label = f.Label
		existing.ValueType = f.ValueType
		existing.UseCount++
		existing.LastUsedAt = now
		existing.UpdatedAt = now
		if err := s.db.Save(&existing).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListConsentFieldRecommendations(organizationID string, limit int) ([]domain.ConsentFieldRecommendation, error) {
	if limit <= 0 {
		limit = 10
	}
	var rows []OrganizationConsentFieldModel
	if err := s.db.Where("organization_id = ?", organizationID).
		Order("use_count desc, last_used_at desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ConsentFieldRecommendation, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.ConsentFieldRecommendation{
			Key:       r.FieldKey,
			Label:     r.Label,
			ValueType: r.ValueType,
			UseCount:  r.UseCount,
		})
	}
	return out, nil
}

func mapConsentModel(m ConsentTemplateModel) (domain.ConsentTemplate, bool, error) {
	var fields []domain.ConsentField
	if m.RequiredFieldsJSON != "" {
		_ = json.Unmarshal([]byte(m.RequiredFieldsJSON), &fields)
	}
	if fields == nil {
		fields = []domain.ConsentField{}
	}
	return domain.ConsentTemplate{
		ID:             m.ID,
		EventID:        m.EventID,
		RequiredFields: fields,
		TrackEntryExit: m.TrackEntryExit,
		TrackMovement:  m.TrackMovement,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}, true, nil
}
