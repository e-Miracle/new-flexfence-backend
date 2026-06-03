package mysql

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/notify"
	"github.com/flexfence/flexfence-backend/internal/store"
	"gorm.io/gorm"
)

type Store struct {
	db       *gorm.DB
	notifier *notify.Dispatcher
}

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) SetNotifier(n *notify.Dispatcher) {
	s.notifier = n
}

func (s *Store) CreateEvent(organizationID, createdByID, title, description string, startAt, endAt time.Time, geofenceGpsTolerance string) (domain.Event, error) {
	qrToken, err := auth.GenerateQRToken()
	if err != nil {
		return domain.Event{}, err
	}
	event := EventModel{
		ID:                   fmt.Sprintf("evt_%d", time.Now().UTC().UnixNano()),
		OrganizationID:       organizationID,
		CreatedByID:          createdByID,
		Title:                title,
		Description:          description,
		StartAt:              startAt.UTC(),
		EndAt:                endAt.UTC(),
		Status:               "active",
		QRToken:              qrToken,
		GeofenceGpsTolerance: domain.NormalizeGeofenceGpsTolerance(geofenceGpsTolerance),
		CreatedAt:            time.Now().UTC(),
	}
	if err := s.db.Create(&event).Error; err != nil {
		return domain.Event{}, err
	}
	return mapEventModel(event), nil
}

func (s *Store) ListEvents() ([]domain.Event, error) {
	return s.listEventsQuery(s.db)
}

func (s *Store) ListEventsByOrganization(organizationID string) ([]domain.Event, error) {
	return s.listEventsQuery(s.db.Where("organization_id = ?", organizationID))
}

func (s *Store) listEventsQuery(query *gorm.DB) ([]domain.Event, error) {
	var events []EventModel
	if err := query.Order("created_at desc").Find(&events).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Event, 0, len(events))
	for _, e := range events {
		out = append(out, mapEventModel(e))
	}
	return out, nil
}

func (s *Store) GetEvent(eventID string) (domain.Event, bool, error) {
	var event EventModel
	err := s.db.Where("id = ?", eventID).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Event{}, false, nil
	}
	if err != nil {
		return domain.Event{}, false, err
	}
	return mapEventModel(event), true, nil
}

func (s *Store) GetEventForOrganization(eventID, organizationID string) (domain.Event, bool, error) {
	var event EventModel
	err := s.db.Where("id = ? AND organization_id = ?", eventID, organizationID).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Event{}, false, nil
	}
	if err != nil {
		return domain.Event{}, false, err
	}
	return mapEventModel(event), true, nil
}

func (s *Store) AddFence(eventID string, in domain.FenceCreateInput) (domain.Fence, error) {
	_, ok, err := s.GetEvent(eventID)
	if err != nil {
		return domain.Fence{}, err
	}
	if !ok {
		return domain.Fence{}, store.ErrEventNotFound
	}
	fence := FenceModel{
		ID:          fmt.Sprintf("fence_%d", time.Now().UTC().UnixNano()),
		EventID:     eventID,
		Name:        in.Name,
		ShapeType:   in.ShapeType,
		StartAt:     in.StartAt.UTC(),
		EndAt:       in.EndAt.UTC(),
		CenterLat:   in.CenterLat,
		CenterLng:   in.CenterLng,
		RadiusM:     in.RadiusM,
		PolygonJSON: in.PolygonJSON,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.db.Create(&fence).Error; err != nil {
		return domain.Fence{}, err
	}
	return mapFenceModel(fence), nil
}

func (s *Store) EnsureEventQRToken(eventID string) (string, error) {
	var event EventModel
	if err := s.db.Where("id = ?", eventID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", store.ErrEventNotFound
		}
		return "", err
	}
	if strings.TrimSpace(event.QRToken) != "" {
		return event.QRToken, nil
	}
	token, err := auth.GenerateQRToken()
	if err != nil {
		return "", err
	}
	if err := s.db.Model(&event).Update("qr_token", token).Error; err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) RegenerateEventQRToken(eventID string) (string, error) {
	if _, ok, err := s.GetEvent(eventID); err != nil {
		return "", err
	} else if !ok {
		return "", store.ErrEventNotFound
	}
	token, err := auth.GenerateQRToken()
	if err != nil {
		return "", err
	}
	if err := s.db.Model(&EventModel{}).Where("id = ?", eventID).Update("qr_token", token).Error; err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) JoinByQR(eventID, userID, qrToken string) (domain.EventJoin, error) {
	if strings.TrimSpace(qrToken) == "" {
		return domain.EventJoin{}, store.ErrInvalidQRToken
	}
	if _, ok, err := s.GetEvent(eventID); err != nil {
		return domain.EventJoin{}, err
	} else if !ok {
		return domain.EventJoin{}, store.ErrEventNotFound
	}
	expected, err := s.EnsureEventQRToken(eventID)
	if err != nil {
		return domain.EventJoin{}, err
	}
	if qrToken != expected {
		return domain.EventJoin{}, store.ErrInvalidQRToken
	}

	var existing EventJoinModel
	err = s.db.Where("event_id = ? AND user_id = ?", eventID, userID).First(&existing).Error
	if err == nil {
		return mapJoinModel(existing), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.EventJoin{}, err
	}

	join := EventJoinModel{
		ID:         fmt.Sprintf("join_%d", time.Now().UTC().UnixNano()),
		EventID:    eventID,
		UserID:     userID,
		JoinSource: "qr",
		QRToken:    qrToken,
		JoinedAt:   time.Now().UTC(),
	}
	if err := s.db.Create(&join).Error; err != nil {
		return domain.EventJoin{}, err
	}
	return mapJoinModel(join), nil
}

func (s *Store) MarkPresent(eventID, userID, source string, lat, lng, accuracyM float64) (domain.AttendanceRecord, error) {
	_, ok, err := s.GetEvent(eventID)
	if err != nil {
		return domain.AttendanceRecord{}, err
	}
	if !ok {
		return domain.AttendanceRecord{}, store.ErrEventNotFound
	}

	var existing AttendanceModel
	err = s.db.Where("event_id = ? AND user_id = ? AND status = ?", eventID, userID, "present").First(&existing).Error
	if err == nil {
		return domain.AttendanceRecord{}, store.ErrAlreadyMarked
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.AttendanceRecord{}, err
	}

	markedAt := time.Now().UTC()
	fenceID := ""
	fences, _ := s.ListFencesByEvent(eventID)
	if len(fences) > 0 {
		fenceID = domain.ResolveFenceForPoint(fences, lat, lng, markedAt)
	}

	record := AttendanceModel{
		ID:        fmt.Sprintf("att_%d", markedAt.UnixNano()),
		EventID:   eventID,
		FenceID:   fenceID,
		UserID:    userID,
		Status:    "present",
		Source:    source,
		MarkedAt:  markedAt,
		Lat:       lat,
		Lng:       lng,
		AccuracyM: accuracyM,
	}
	if err := s.db.Create(&record).Error; err != nil {
		return domain.AttendanceRecord{}, err
	}
	return mapAttendanceModel(record), nil
}

func mapEventModel(m EventModel) domain.Event {
	return domain.Event{
		ID:             m.ID,
		OrganizationID: m.OrganizationID,
		CreatedByID:    m.CreatedByID,
		Title:          m.Title,
		Description:    m.Description,
		StartAt:        m.StartAt,
		EndAt:          m.EndAt,
		Status:         m.Status,
		CreatedAt:                m.CreatedAt,
		QRToken:                  m.QRToken,
		ScanToClockInEnabled:     m.ScanToClockInEnabled,
		ClockInQRToken:           m.ClockInQRToken,
		ClockInQRIssuedAt:        m.ClockInQRIssuedAt,
		ClockInQRRotationMinutes: m.ClockInQRRotationMinutes,
		GeofenceGpsTolerance:     domain.NormalizeGeofenceGpsTolerance(m.GeofenceGpsTolerance),
	}
}

func mapFenceModel(m FenceModel) domain.Fence {
	return domain.Fence{
		ID:          m.ID,
		EventID:     m.EventID,
		Name:        m.Name,
		ShapeType:   m.ShapeType,
		StartAt:     m.StartAt,
		EndAt:       m.EndAt,
		CenterLat:   m.CenterLat,
		CenterLng:   m.CenterLng,
		RadiusM:     m.RadiusM,
		PolygonJSON: m.PolygonJSON,
		CreatedAt:   m.CreatedAt,
	}
}

func mapJoinModel(m EventJoinModel) domain.EventJoin {
	return domain.EventJoin{
		ID:         m.ID,
		EventID:    m.EventID,
		UserID:     m.UserID,
		JoinSource: m.JoinSource,
		QRToken:    m.QRToken,
		JoinedAt:   m.JoinedAt,
	}
}

func mapAttendanceModel(m AttendanceModel) domain.AttendanceRecord {
	return domain.AttendanceRecord{
		ID:        m.ID,
		EventID:   m.EventID,
		FenceID:   m.FenceID,
		UserID:    m.UserID,
		Status:    m.Status,
		Source:    m.Source,
		MarkedAt:  m.MarkedAt,
		Lat:       m.Lat,
		Lng:       m.Lng,
		AccuracyM: m.AccuracyM,
	}
}
