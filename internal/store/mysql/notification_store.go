package mysql

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
	"gorm.io/gorm"
)

func (s *IdentityStore) GetUserNotificationPreferences(userID string) (domain.UserNotificationPreferences, error) {
	var user UserModel
	err := s.db.Where("id = ?", userID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.UserNotificationPreferences{}, store.ErrUserNotFound
	}
	if err != nil {
		return domain.UserNotificationPreferences{}, err
	}
	return mapNotificationPrefs(user), nil
}

func (s *IdentityStore) UpdateUserNotificationPreferences(userID string, prefs domain.UserNotificationPreferences) (domain.UserNotificationPreferences, error) {
	res := s.db.Model(&UserModel{}).Where("id = ?", userID).Updates(map[string]any{
		"notify_geofence":    prefs.GeofenceNotificationsEnabled,
		"notify_missed_check": prefs.MissedCheckInOutEnabled,
		"notify_sound":       prefs.SoundAndVibrationEnabled,
		"notify_email":       prefs.EmailNotificationsEnabled,
		"notify_push":        prefs.PushNotificationsEnabled,
		"updated_at":         time.Now().UTC(),
	})
	if res.Error != nil {
		return domain.UserNotificationPreferences{}, res.Error
	}
	if res.RowsAffected == 0 {
		return domain.UserNotificationPreferences{}, store.ErrUserNotFound
	}
	return s.GetUserNotificationPreferences(userID)
}

func (s *IdentityStore) UpsertUserDeviceToken(userID, platform, token string) error {
	token = strings.TrimSpace(token)
	platform = strings.TrimSpace(strings.ToLower(platform))
	if token == "" || platform == "" {
		return fmt.Errorf("token and platform are required")
	}
	now := time.Now().UTC()
	var existing UserDeviceTokenModel
	err := s.db.Where("token = ?", token).First(&existing).Error
	if err == nil {
		return s.db.Model(&existing).Updates(map[string]any{
			"user_id":    userID,
			"platform":   platform,
			"updated_at": now,
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	row := UserDeviceTokenModel{
		ID:        fmt.Sprintf("dt_%d", now.UnixNano()),
		UserID:    userID,
		Platform:  platform,
		Token:     token,
		UpdatedAt: now,
	}
	return s.db.Create(&row).Error
}

func (s *IdentityStore) DeleteUserDeviceToken(userID, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return s.db.Where("user_id = ? AND token = ?", userID, token).Delete(&UserDeviceTokenModel{}).Error
}

func (s *IdentityStore) ListUserDeviceTokens(userID string) ([]string, error) {
	var rows []UserDeviceTokenModel
	if err := s.db.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Token) != "" {
			out = append(out, row.Token)
		}
	}
	return out, nil
}

func mapNotificationPrefs(m UserModel) domain.UserNotificationPreferences {
	return domain.UserNotificationPreferences{
		GeofenceNotificationsEnabled: m.NotifyGeofence,
		MissedCheckInOutEnabled:      m.NotifyMissedCheck,
		SoundAndVibrationEnabled:     m.NotifySound,
		EmailNotificationsEnabled:    m.NotifyEmail,
		PushNotificationsEnabled:     m.NotifyPush,
	}
}
