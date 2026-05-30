package memory

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
)

type Store struct {
	mu sync.RWMutex

	eventSeq      int64
	fenceSeq      int64
	joinSeq       int64
	attendanceSeq int64
	consentSeq    int64

	events      map[string]domain.Event
	fences      map[string][]domain.Fence
	joins       map[string][]domain.EventJoin
	attendance  map[string][]domain.AttendanceRecord
	consent     map[string]domain.ConsentTemplate
	orgConsent  map[string]map[string]domain.ConsentFieldRecommendation // orgID -> fieldKey
	joinByUser  map[string]map[string]bool // event_id -> user_id -> joined
	presentUser map[string]map[string]bool // event_id -> user_id -> present
}

func NewStore() *Store {
	return &Store{
		events:      make(map[string]domain.Event),
		fences:      make(map[string][]domain.Fence),
		joins:       make(map[string][]domain.EventJoin),
		attendance:  make(map[string][]domain.AttendanceRecord),
		consent:     make(map[string]domain.ConsentTemplate),
		orgConsent:  make(map[string]map[string]domain.ConsentFieldRecommendation),
		joinByUser:  make(map[string]map[string]bool),
		presentUser: make(map[string]map[string]bool),
	}
}

func (s *Store) CreateEvent(organizationID, createdByID, title, description string, startAt, endAt time.Time) (domain.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.eventSeq++
	id := fmt.Sprintf("evt_%d", s.eventSeq)
	now := time.Now().UTC()
	qrToken, err := auth.GenerateQRToken()
	if err != nil {
		return domain.Event{}, err
	}
	event := domain.Event{
		ID:             id,
		OrganizationID: organizationID,
		CreatedByID:    createdByID,
		Title:          title,
		Description:    description,
		StartAt:        startAt.UTC(),
		EndAt:          endAt.UTC(),
		Status:         "active",
		CreatedAt:      now,
		QRToken:        qrToken,
	}
	s.events[id] = event
	return event, nil
}

func (s *Store) EnsureEventQRToken(eventID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.events[eventID]
	if !ok {
		return "", store.ErrEventNotFound
	}
	if strings.TrimSpace(event.QRToken) != "" {
		return event.QRToken, nil
	}
	token, err := auth.GenerateQRToken()
	if err != nil {
		return "", err
	}
	event.QRToken = token
	s.events[eventID] = event
	return token, nil
}

func (s *Store) RegenerateEventQRToken(eventID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.events[eventID]
	if !ok {
		return "", store.ErrEventNotFound
	}
	token, err := auth.GenerateQRToken()
	if err != nil {
		return "", err
	}
	event.QRToken = token
	s.events[eventID] = event
	return token, nil
}

func (s *Store) ListEvents() ([]domain.Event, error) {
	return s.ListEventsByOrganization("")
}

func (s *Store) ListEventsByOrganization(organizationID string) ([]domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.Event, 0, len(s.events))
	for _, event := range s.events {
		if organizationID == "" || event.OrganizationID == organizationID {
			out = append(out, event)
		}
	}
	return out, nil
}

func (s *Store) GetEvent(eventID string) (domain.Event, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.events[eventID]
	return event, ok, nil
}

func (s *Store) GetEventForOrganization(eventID, organizationID string) (domain.Event, bool, error) {
	event, ok, err := s.GetEvent(eventID)
	if err != nil || !ok {
		return event, ok, err
	}
	if event.OrganizationID != organizationID {
		return domain.Event{}, false, nil
	}
	return event, true, nil
}

func (s *Store) AddFence(eventID string, in domain.FenceCreateInput) (domain.Fence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[eventID]; !ok {
		return domain.Fence{}, store.ErrEventNotFound
	}
	s.fenceSeq++
	fence := domain.Fence{
		ID:          fmt.Sprintf("fence_%d", s.fenceSeq),
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
	s.fences[eventID] = append(s.fences[eventID], fence)
	return fence, nil
}

func (s *Store) DeleteFence(eventID, fenceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, ok := s.events[eventID]
	if !ok {
		return store.ErrEventNotFound
	}
	if domain.EventIsLive(event, time.Now().UTC()) {
		return store.ErrEventLive
	}
	fences := s.fences[eventID]
	next := make([]domain.Fence, 0, len(fences))
	found := false
	for _, f := range fences {
		if f.ID == fenceID {
			found = true
			continue
		}
		next = append(next, f)
	}
	if !found {
		return store.ErrFenceNotFound
	}
	s.fences[eventID] = next
	return nil
}

func (s *Store) ProcessPendingEventGoLive(now time.Time) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	processed := 0
	for id, event := range s.events {
		if !domain.EventIsLive(event, now) {
			continue
		}
		_ = id
		processed++
	}
	return processed, nil
}

func (s *Store) ConsumeUserGeofenceAlerts(userID string) ([]domain.GeofenceAlert, error) {
	return []domain.GeofenceAlert{}, nil
}

func (s *Store) JoinByQR(eventID, userID, qrToken string) (domain.EventJoin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[eventID]; !ok {
		return domain.EventJoin{}, store.ErrEventNotFound
	}
	if strings.TrimSpace(qrToken) == "" {
		return domain.EventJoin{}, store.ErrInvalidQRToken
	}
	event := s.events[eventID]
	if strings.TrimSpace(event.QRToken) == "" {
		token, err := auth.GenerateQRToken()
		if err != nil {
			return domain.EventJoin{}, err
		}
		event.QRToken = token
		s.events[eventID] = event
	}
	if qrToken != event.QRToken {
		return domain.EventJoin{}, store.ErrInvalidQRToken
	}

	if s.joinByUser[eventID] == nil {
		s.joinByUser[eventID] = make(map[string]bool)
	}
	if s.joinByUser[eventID][userID] {
		// idempotent behavior; return a synthetic state as joined.
		return domain.EventJoin{
			ID:         "",
			EventID:    eventID,
			UserID:     userID,
			JoinSource: "qr",
			QRToken:    qrToken,
			JoinedAt:   time.Now().UTC(),
		}, nil
	}

	s.joinSeq++
	join := domain.EventJoin{
		ID:         fmt.Sprintf("join_%d", s.joinSeq),
		EventID:    eventID,
		UserID:     userID,
		JoinSource: "qr",
		QRToken:    qrToken,
		JoinedAt:   time.Now().UTC(),
	}
	s.joins[eventID] = append(s.joins[eventID], join)
	s.joinByUser[eventID][userID] = true
	return join, nil
}

func (s *Store) MarkPresent(eventID, userID, source string, lat, lng, accuracyM float64) (domain.AttendanceRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[eventID]; !ok {
		return domain.AttendanceRecord{}, store.ErrEventNotFound
	}

	if s.presentUser[eventID] == nil {
		s.presentUser[eventID] = make(map[string]bool)
	}
	if s.presentUser[eventID][userID] {
		return domain.AttendanceRecord{}, store.ErrAlreadyMarked
	}

	markedAt := time.Now().UTC()
	fences := s.fences[eventID]
	fenceID := domain.ResolveFenceForPoint(fences, lat, lng, markedAt)

	s.attendanceSeq++
	record := domain.AttendanceRecord{
		ID:        fmt.Sprintf("att_%d", s.attendanceSeq),
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
	s.attendance[eventID] = append(s.attendance[eventID], record)
	s.presentUser[eventID][userID] = true
	return record, nil
}

func (s *Store) ListFencesByEvent(eventID string) ([]domain.Fence, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.Fence(nil), s.fences[eventID]...), nil
}

func (s *Store) ListAttendanceByEvent(eventID string) ([]domain.AttendanceWithUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := s.attendance[eventID]
	out := make([]domain.AttendanceWithUser, 0, len(records))
	for _, r := range records {
		out = append(out, domain.AttendanceWithUser{
			ID: r.ID, EventID: r.EventID, FenceID: r.FenceID, UserID: r.UserID,
			Status: r.Status, Source: r.Source, MarkedAt: r.MarkedAt,
			Lat: r.Lat, Lng: r.Lng, AccuracyM: r.AccuracyM,
		})
	}
	return out, nil
}

func (s *Store) GetFenceByEvent(eventID, fenceID string) (domain.Fence, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, f := range s.fences[eventID] {
		if f.ID == fenceID {
			return f, true, nil
		}
	}
	return domain.Fence{}, false, nil
}

func (s *Store) ListJoinsByEvent(eventID string) ([]domain.EventJoin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.EventJoin(nil), s.joins[eventID]...), nil
}

func (s *Store) ListJoinsByUser(userID string) ([]domain.UserEventJoin, error) {
	page, err := s.ListJoinsByUserFiltered(userID, store.UserEventJoinFilter{Page: 1, Limit: 1000})
	if err != nil {
		return nil, err
	}
	return page.Joins, nil
}

func (s *Store) ListJoinsByUserFiltered(userID string, filter store.UserEventJoinFilter) (store.UserEventJoinPage, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	s.mu.RLock()
	all := s.collectUserJoinsLocked(userID)
	s.mu.RUnlock()

	filtered := make([]domain.UserEventJoin, 0, len(all))
	for _, j := range all {
		if filter.Search != "" && !stringsContainsFold(j.EventTitle, filter.Search) &&
			!stringsContainsFold(j.EventDescription, filter.Search) {
			continue
		}
		if filter.Status != "" && j.EventStatus != filter.Status {
			continue
		}
		if filter.JoinSource != "" && j.JoinSource != filter.JoinSource {
			continue
		}
		if filter.StartFrom != nil && (j.EventStartAt.IsZero() || j.EventStartAt.Before(*filter.StartFrom)) {
			continue
		}
		if filter.StartTo != nil && (j.EventStartAt.IsZero() || j.EventStartAt.After(*filter.StartTo)) {
			continue
		}
		filtered = append(filtered, j)
	}

	sortUserJoinsByEventDate(filtered)

	total := len(filtered)
	offset := (filter.Page - 1) * filter.Limit
	if offset > total {
		offset = total
	}
	end := offset + filter.Limit
	if end > total {
		end = total
	}
	pageItems := filtered[offset:end]

	hasMore := end < total
	return store.UserEventJoinPage{
		Joins:   pageItems,
		Total:   total,
		Page:    filter.Page,
		Limit:   filter.Limit,
		HasMore: hasMore,
	}, nil
}

func (s *Store) DeleteUserEventJoin(userID, joinID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for eventID, joins := range s.joins {
		for i, j := range joins {
			if j.ID == joinID && j.UserID == userID {
				s.joins[eventID] = append(joins[:i], joins[i+1:]...)
				return nil
			}
		}
	}
	return store.ErrJoinNotFound
}

func (s *Store) collectUserJoinsLocked(userID string) []domain.UserEventJoin {
	var out []domain.UserEventJoin
	for eventID, joins := range s.joins {
		event, ok, _ := s.getEventLocked(eventID)
		for _, j := range joins {
			if j.UserID != userID {
				continue
			}
			title := eventID
			description := ""
			status := ""
			var startAt, endAt time.Time
			if ok {
				title = event.Title
				description = event.Description
				status = event.Status
				startAt = event.StartAt
				endAt = event.EndAt
			}
			out = append(out, domain.UserEventJoin{
				ID:               j.ID,
				EventID:          j.EventID,
				EventTitle:       title,
				EventDescription: description,
				EventStartAt:     startAt,
				EventEndAt:       endAt,
				EventStatus:      status,
				JoinSource:       j.JoinSource,
				JoinedAt:         j.JoinedAt,
			})
		}
	}
	return out
}

func sortUserJoinsByEventDate(joins []domain.UserEventJoin) {
	for i := 0; i < len(joins); i++ {
		for j := i + 1; j < len(joins); j++ {
			if joins[j].EventStartAt.After(joins[i].EventStartAt) {
				joins[i], joins[j] = joins[j], joins[i]
			}
		}
	}
}

func stringsContainsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func (s *Store) getEventLocked(eventID string) (domain.Event, bool, error) {
	event, ok := s.events[eventID]
	return event, ok, nil
}

func (s *Store) GetEventAnalytics(eventID string) (domain.EventAnalytics, error) {
	event, ok, err := s.GetEvent(eventID)
	if err != nil {
		return domain.EventAnalytics{}, err
	}
	if !ok {
		return domain.EventAnalytics{}, store.ErrEventNotFound
	}
	fences, err := s.ListFencesByEvent(eventID)
	if err != nil {
		return domain.EventAnalytics{}, err
	}
	joins, err := s.ListJoinsByEvent(eventID)
	if err != nil {
		return domain.EventAnalytics{}, err
	}
	attendance, err := s.ListAttendanceByEvent(eventID)
	if err != nil {
		return domain.EventAnalytics{}, err
	}
	return domain.ComputeEventAnalytics(domain.AnalyticsInput{
		Event: event, Fences: fences, Joins: joins, Attendance: attendance,
	}), nil
}

func (s *Store) GetFenceAnalytics(eventID, fenceID string) (domain.FenceAnalytics, error) {
	event, ok, err := s.GetEvent(eventID)
	if err != nil {
		return domain.FenceAnalytics{}, err
	}
	if !ok {
		return domain.FenceAnalytics{}, store.ErrEventNotFound
	}
	fence, ok, err := s.GetFenceByEvent(eventID, fenceID)
	if err != nil {
		return domain.FenceAnalytics{}, err
	}
	if !ok {
		return domain.FenceAnalytics{}, store.ErrEventNotFound
	}
	fences, _ := s.ListFencesByEvent(eventID)
	attendance, _ := s.ListAttendanceByEvent(eventID)
	return domain.ComputeFenceAnalytics(fence, event, fences, attendance), nil
}

func (s *Store) GetConsentTemplate(eventID string) (domain.ConsentTemplate, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tpl, ok := s.consent[eventID]
	return tpl, ok, nil
}

func (s *Store) SaveConsentTemplate(eventID string, tpl domain.ConsentTemplate) (domain.ConsentTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.events[eventID]; !ok {
		return domain.ConsentTemplate{}, store.ErrEventNotFound
	}
	now := time.Now().UTC()
	tpl.EventID = eventID
	tpl.UpdatedAt = now
	if existing, ok := s.consent[eventID]; ok {
		tpl.ID = existing.ID
		tpl.CreatedAt = existing.CreatedAt
	} else {
		s.consentSeq++
		tpl.ID = fmt.Sprintf("consent_%d", s.consentSeq)
		tpl.CreatedAt = now
	}
	s.consent[eventID] = tpl
	return tpl, nil
}

func (s *Store) RecordOrganizationConsentFields(organizationID string, fields []domain.ConsentField) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.orgConsent[organizationID] == nil {
		s.orgConsent[organizationID] = make(map[string]domain.ConsentFieldRecommendation)
	}
	now := time.Now().UTC()
	for _, f := range fields {
		if !f.IsCustom {
			continue
		}
		if existing, ok := s.orgConsent[organizationID][f.Key]; ok {
			existing.Label = f.Label
			existing.ValueType = f.ValueType
			existing.UseCount++
			s.orgConsent[organizationID][f.Key] = existing
		} else {
			s.orgConsent[organizationID][f.Key] = domain.ConsentFieldRecommendation{
				Key:       f.Key,
				Label:     f.Label,
				ValueType: f.ValueType,
				UseCount:  1,
			}
		}
		_ = now
	}
	return nil
}

func (s *Store) ListConsentFieldRecommendations(organizationID string, limit int) ([]domain.ConsentFieldRecommendation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 10
	}
	bucket := s.orgConsent[organizationID]
	out := make([]domain.ConsentFieldRecommendation, 0, len(bucket))
	for _, r := range bucket {
		out = append(out, r)
	}
	// Simple sort by use_count desc
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].UseCount > out[i].UseCount {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) ListSubscribedGeofenceEvents(userID string, now time.Time) ([]domain.SubscribedGeofenceEvent, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	joins, err := s.ListJoinsByUser(userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.SubscribedGeofenceEvent, 0, len(joins))
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, join := range joins {
		if !join.EventEndAt.IsZero() && join.EventEndAt.Before(now) {
			continue
		}
		fences := s.fences[join.EventID]
		circleFences := make([]domain.Fence, 0, len(fences))
		for _, f := range fences {
			if f.ShapeType != "circle" || f.RadiusM <= 0 {
				continue
			}
			if f.CenterLat == 0 && f.CenterLng == 0 {
				continue
			}
			circleFences = append(circleFences, f)
		}
		out = append(out, domain.SubscribedGeofenceEvent{
			ID:               join.ID,
			EventID:          join.EventID,
			EventTitle:       join.EventTitle,
			EventDescription: join.EventDescription,
			EventStartAt:     join.EventStartAt,
			EventEndAt:       join.EventEndAt,
			EventStatus:      join.EventStatus,
			JoinSource:       join.JoinSource,
			JoinedAt:         join.JoinedAt,
			Fences:           circleFences,
		})
	}
	return out, nil
}

func (s *Store) ListUserActivityHistory(userID string, filter store.ActivityHistoryFilter) ([]domain.UserActivitySession, error) {
	return []domain.UserActivitySession{}, nil
}

func (s *Store) RecordClockIn(userID, eventID, eventTitle, fenceID, fenceName, source string, lat, lng float64) (domain.UserActivitySession, error) {
	now := time.Now().UTC()
	return domain.UserActivitySession{
		ID:         fmt.Sprintf("act_%d", now.UnixNano()),
		UserID:     userID,
		EventID:    eventID,
		EventTitle: eventTitle,
		FenceID:    fenceID,
		FenceName:  fenceName,
		ClockInAt:  now,
		Verified:   source == "geofence",
		Source:     source,
		CreatedAt:  now,
	}, nil
}

func (s *Store) RecordClockOut(userID, sessionID, eventID, fenceID string) (domain.UserActivitySession, error) {
	return domain.UserActivitySession{}, store.ErrOpenSessionNotFound
}

func (s *Store) DeleteActivityHistoryOlderThan(cutoff time.Time) (int64, error) {
	return 0, nil
}

func (s *Store) CreateFenceCaptureSession(eventID, targetShape string) (domain.FenceCaptureSession, error) {
	return domain.FenceCaptureSession{}, store.ErrEventNotFound
}

func (s *Store) GetActiveFenceCaptureSession(eventID string) (domain.FenceCaptureSession, bool, error) {
	return domain.FenceCaptureSession{}, false, nil
}

func (s *Store) GetFenceCaptureSessionByToken(token string) (domain.FenceCaptureSession, bool, error) {
	return domain.FenceCaptureSession{}, false, nil
}

func (s *Store) AppendFenceCapturePoint(token string, point domain.FenceCapturePoint) (domain.FenceCaptureSession, error) {
	return domain.FenceCaptureSession{}, store.ErrCaptureNotFound
}

func (s *Store) ApplyFenceCaptureSession(eventID, sessionID, name string, startAt, endAt time.Time) (domain.Fence, error) {
	return domain.Fence{}, store.ErrCaptureNotFound
}

