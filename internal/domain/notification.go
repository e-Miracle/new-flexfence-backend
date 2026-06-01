package domain

// UserNotificationPreferences mirrors mobile notification settings.
type UserNotificationPreferences struct {
	GeofenceNotificationsEnabled bool `json:"geofence_notifications_enabled"`
	MissedCheckInOutEnabled      bool `json:"missed_check_in_out_enabled"`
	SoundAndVibrationEnabled     bool `json:"sound_and_vibration_enabled"`
	EmailNotificationsEnabled    bool `json:"email_notifications_enabled"`
	PushNotificationsEnabled     bool `json:"push_notifications_enabled"`
}
