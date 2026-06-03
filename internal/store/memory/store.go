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
	userConsentSeq int64

	events      map[string]domain.Event
	fences      map[string][]domain.Fence
	joins       map[string][]domain.EventJoin
	attendance  map[string][]domain.AttendanceRecord
	consent     map[string]domain.ConsentTemplate
	userConsents map[string]domain.UserConsent // key: eventID+"\x00"+userID
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
		userConsents: make(map[string]domain.UserConsent),
		orgConsent:  make(map[string]map[string]domain.ConsentFieldRecommendation),
		joinByUser:  make(map[string]map[string]bool),
		presentUser: make(map[string]map[string]bool),
	}
}

func (s *Store) CreateEvent(organizationID, createdByID, title, description string, startAt, endAt time.Time, geofenceGpsTolerance string) (domain.Event, error) {
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
		ID:                   id,
		OrganizationID:       organizationID,
		CreatedByID:          createdByID,
		Title:                title,
		Description:          description,
		StartAt:              startAt.UTC(),
		EndAt:                endAt.UTC(),
		Status:               "active",
		CreatedAt:            now,
		QRToken:              qrToken,
		GeofenceGpsTolerance: domain.NormalizeGeofenceGpsTolerance(geofenceGpsTolerance),
	}
	s.events[id] = event
	return event, nil
}

func (s *Store) UpdateEvent(
	eventID, organizationID string,
	title, description string,
	startAt, endAt time.Time,
	geofenceGpsTolerance string,
) (domain.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, ok := s.events[eventID]
	if !ok || event.OrganizationID != organizationID {
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
	updated := event
	updated.Title = title
	updated.Description = strings.TrimSpace(description)
	updated.StartAt = startAt
	updated.EndAt = endAt
	if strings.TrimSpace(geofenceGpsTolerance) != "" {
		updated.GeofenceGpsTolerance = domain.NormalizeGeofenceGpsTolerance(geofenceGpsTolerance)
	}
	for _, fence := range s.fences[eventID] {
		fs := fence.StartAt
		fe := fence.EndAt
		if _, _, err := domain.ResolveFenceSchedule(updated, &fs, &fe); err != nil {
			return domain.Event{}, fmt.Errorf(
				"%w: fence %q does not fit the updated schedule (%v)",
				store.ErrInvalidSchedule,
				fence.Name,
				err,
			)
		}
	}
	s.events[eventID] = updated
	return updated, nil
}

func (s *Store) DeleteEvent(eventID, organizationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, ok := s.events[eventID]
	if !ok || event.OrganizationID != organizationID {
		return store.ErrEventNotFound
	}
	if domain.EventIsLive(event, time.Now().UTC()) {
		return store.ErrEventLive
	}
	delete(s.events, eventID)
	delete(s.fences, eventID)
	delete(s.joins, eventID)
	delete(s.attendance, eventID)
	delete(s.consent, eventID)
	for key := range s.userConsents {
		if strings.HasPrefix(key, eventID+"\x00") {
			delete(s.userConsents, key)
		}
	}
	delete(s.joinByUser, eventID)
	delete(s.presentUser, eventID)
	return nil
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

func (s *Store) UpdateEventClockInSettings(eventID string, enabled bool, rotationMinutes int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.events[eventID]
	if !ok {
		return store.ErrEventNotFound
	}
	if rotationMinutes < 0 || (rotationMinutes > 0 && rotationMinutes < 5) {
		return store.ErrInvalidSchedule
	}
	event.ScanToClockInEnabled = enabled
	event.ClockInQRRotationMinutes = rotationMinutes
	if !enabled {
		event.ClockInQRToken = ""
		event.ClockInQRIssuedAt = nil
	}
	s.events[eventID] = event
	if enabled && strings.TrimSpace(event.ClockInQRToken) == "" {
		_, _, err := s.ensureEventClockInQRTokenLocked(eventID)
		return err
	}
	return nil
}

func (s *Store) EnsureEventClockInQRToken(eventID string) (string, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ensureEventClockInQRTokenLocked(eventID)
}

func (s *Store) ensureEventClockInQRTokenLocked(eventID string) (string, time.Time, error) {
	event, ok := s.events[eventID]
	if !ok {
		return "", time.Time{}, store.ErrEventNotFound
	}
	if !event.ScanToClockInEnabled {
		return "", time.Time{}, store.ErrClockInQRDisabled
	}
	now := time.Now().UTC()
	if strings.TrimSpace(event.ClockInQRToken) != "" &&
		!memoryClockInQRIsExpired(event.ClockInQRIssuedAt, event.ClockInQRRotationMinutes, now) {
		issued := now
		if event.ClockInQRIssuedAt != nil {
			issued = event.ClockInQRIssuedAt.UTC()
		}
		return event.ClockInQRToken, issued, nil
	}
	return s.issueEventClockInQRTokenLocked(eventID, now)
}

func (s *Store) RegenerateEventClockInQRToken(eventID string) (string, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.events[eventID]
	if !ok {
		return "", time.Time{}, store.ErrEventNotFound
	}
	if !event.ScanToClockInEnabled {
		return "", time.Time{}, store.ErrClockInQRDisabled
	}
	return s.issueEventClockInQRTokenLocked(eventID, time.Now().UTC())
}

func (s *Store) issueEventClockInQRTokenLocked(eventID string, now time.Time) (string, time.Time, error) {
	token, err := auth.GenerateQRToken()
	if err != nil {
		return "", time.Time{}, err
	}
	event := s.events[eventID]
	issuedAt := now.UTC()
	event.ClockInQRToken = token
	event.ClockInQRIssuedAt = &issuedAt
	s.events[eventID] = event
	return token, issuedAt, nil
}

func memoryClockInQRIsExpired(issuedAt *time.Time, rotationMinutes int, now time.Time) bool {
	if rotationMinutes <= 0 || issuedAt == nil || issuedAt.IsZero() {
		return false
	}
	return !now.Before(issuedAt.Add(time.Duration(rotationMinutes) * time.Minute))
}

func (s *Store) ValidateEventClockInQRToken(eventID, qrToken string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.events[eventID]
	if !ok {
		return store.ErrEventNotFound
	}
	if !event.ScanToClockInEnabled {
		return store.ErrClockInQRDisabled
	}
	if strings.TrimSpace(qrToken) == "" || qrToken != event.ClockInQRToken {
		return store.ErrInvalidQRToken
	}
	if memoryClockInQRIsExpired(event.ClockInQRIssuedAt, event.ClockInQRRotationMinutes, time.Now().UTC()) {
		return store.ErrClockInQRExpired
	}
	return nil
}

func (s *Store) UserHasJoinedEvent(eventID, userID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, join := range s.joins[eventID] {
		if join.UserID == userID {
			return true, nil
		}
	}
	return false, nil
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

func userConsentKey(eventID, userID string) string {
	return eventID + "\x00" + userID
}

func (s *Store) EventScanToClockInEnabled(eventID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.events[eventID]
	if !ok {
		return false, store.ErrEventNotFound
	}
	return event.ScanToClockInEnabled, nil
}

func (s *Store) UserHasEventConsent(eventID, userID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.userConsents[userConsentKey(eventID, userID)]
	return ok, nil
}

func (s *Store) GetUserEventConsent(eventID, userID string) (domain.UserConsent, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	consent, ok := s.userConsents[userConsentKey(eventID, userID)]
	return consent, ok, nil
}

func (s *Store) SaveUserEventConsent(
	eventID, userID string,
	values map[string]string,
	tpl domain.ConsentTemplate,
) (domain.UserConsent, error) {
	if err := domain.ValidateConsentSubmission(tpl.RequiredFields, values); err != nil {
		return domain.UserConsent{}, fmt.Errorf("%w: %v", store.ErrInvalidConsent, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userConsentKey(eventID, userID)
	snapshot := make(map[string]string, len(values))
	for k, v := range values {
		snapshot[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	now := time.Now().UTC()
	if existing, ok := s.userConsents[key]; ok {
		existing.ConsentSnapshot = snapshot
		existing.AgreedAt = now
		s.userConsents[key] = existing
		return existing, nil
	}
	s.userConsentSeq++
	consent := domain.UserConsent{
		ID:              fmt.Sprintf("uconsent_%d", s.userConsentSeq),
		EventID:         eventID,
		UserID:          userID,
		ConsentSnapshot: snapshot,
		AgreedAt:        now,
	}
	s.userConsents[key] = consent
	return consent, nil
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
		event, ok := s.events[join.EventID]
		scanEnabled := ok && event.ScanToClockInEnabled
		tolerance := domain.GeofenceGpsToleranceDefault
		if ok {
			tolerance = domain.NormalizeGeofenceGpsTolerance(event.GeofenceGpsTolerance)
		}
		out = append(out, domain.SubscribedGeofenceEvent{
			ID:                   join.ID,
			EventID:              join.EventID,
			EventTitle:           join.EventTitle,
			EventDescription:     join.EventDescription,
			EventStartAt:         join.EventStartAt,
			EventEndAt:           join.EventEndAt,
			EventStatus:          join.EventStatus,
			JoinSource:           join.JoinSource,
			JoinedAt:             join.JoinedAt,
			ScanToClockInEnabled: scanEnabled,
			GeofenceGpsTolerance: tolerance,
			Fences:               circleFences,
		})
	}
	return out, nil
}

func (s *Store) ListUserActivityHistory(userID string, filter store.ActivityHistoryFilter) ([]domain.UserActivitySession, error) {
	return []domain.UserActivitySession{}, nil
}

func (s *Store) RecordClockIn(userID, eventID, eventTitle, fenceID, fenceName, source string, lat, lng float64) (domain.UserActivitySession, error) {
	now := time.Now().UTC()
	if strings.TrimSpace(source) == "" {
		source = "manual"
	}
	session := domain.UserActivitySession{
		ID:         fmt.Sprintf("act_%d", now.UnixNano()),
		UserID:     userID,
		EventID:    eventID,
		EventTitle: eventTitle,
		FenceID:    fenceID,
		FenceName:  fenceName,
		ClockInAt:  now,
		Verified:   source == "geofence" || source == "qr_scan" || lat != 0 || lng != 0,
		Source:     source,
		CreatedAt:  now,
	}

	eventID = strings.TrimSpace(eventID)
	if eventID != "" {
		s.mu.Lock()
		defer s.mu.Unlock()

		resolvedFenceID := strings.TrimSpace(fenceID)
		if resolvedFenceID == "" && (lat != 0 || lng != 0) {
			fences := s.fences[eventID]
			if len(fences) > 0 {
				resolvedFenceID = domain.ResolveFenceForPoint(fences, lat, lng, now)
			}
		}
		s.attendanceSeq++
		record := domain.AttendanceRecord{
			ID:       fmt.Sprintf("att_%d", s.attendanceSeq),
			EventID:  eventID,
			FenceID:  resolvedFenceID,
			UserID:   userID,
			Status:   "present",
			Source:   source,
			MarkedAt: now,
			Lat:      lat,
			Lng:      lng,
		}
		s.attendance[eventID] = append(s.attendance[eventID], record)
	}
	return session, nil
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

