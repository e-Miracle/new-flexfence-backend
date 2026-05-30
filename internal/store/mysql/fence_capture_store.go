package mysql

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/auth"
	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
	"gorm.io/gorm"
)

const fenceCaptureTTL = 24 * time.Hour

func normalizeCaptureShape(shape string) string {
	switch strings.TrimSpace(strings.ToLower(shape)) {
	case "circle":
		return "circle"
	default:
		return "polygon"
	}
}

func (s *Store) CreateFenceCaptureSession(eventID, targetShape string) (domain.FenceCaptureSession, error) {
	event, ok, err := s.GetEvent(eventID)
	if err != nil {
		return domain.FenceCaptureSession{}, err
	}
	if !ok {
		return domain.FenceCaptureSession{}, store.ErrEventNotFound
	}

	token, err := auth.GenerateQRToken()
	if err != nil {
		return domain.FenceCaptureSession{}, err
	}
	now := time.Now().UTC()
	shape := normalizeCaptureShape(targetShape)

	_ = s.db.Model(&FenceCaptureSessionModel{}).
		Where("event_id = ? AND status = ?", eventID, "active").
		Updates(map[string]any{"status": "expired"}).Error

	row := FenceCaptureSessionModel{
		ID:          fmt.Sprintf("cap_%d", now.UnixNano()),
		EventID:     eventID,
		Token:       token,
		TargetShape: shape,
		Status:      "active",
		PointsJSON:  "[]",
		ExpiresAt:   now.Add(fenceCaptureTTL),
		CreatedAt:   now,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return domain.FenceCaptureSession{}, err
	}
	return mapCaptureSession(row, event.Title), nil
}

func (s *Store) GetActiveFenceCaptureSession(eventID string) (domain.FenceCaptureSession, bool, error) {
	var row FenceCaptureSessionModel
	err := s.db.Where("event_id = ? AND status = ?", eventID, "active").
		Order("created_at desc").
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return domain.FenceCaptureSession{}, false, nil
	}
	if err != nil {
		return domain.FenceCaptureSession{}, false, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		_ = s.db.Model(&row).Update("status", "expired").Error
		return domain.FenceCaptureSession{}, false, nil
	}
	event, _, _ := s.GetEvent(eventID)
	return mapCaptureSession(row, event.Title), true, nil
}

func (s *Store) GetFenceCaptureSessionByToken(token string) (domain.FenceCaptureSession, bool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.FenceCaptureSession{}, false, nil
	}
	var row FenceCaptureSessionModel
	err := s.db.Where("token = ?", token).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return domain.FenceCaptureSession{}, false, nil
	}
	if err != nil {
		return domain.FenceCaptureSession{}, false, err
	}
	if row.Status != "active" || time.Now().UTC().After(row.ExpiresAt) {
		return domain.FenceCaptureSession{}, false, nil
	}
	event, _, _ := s.GetEvent(row.EventID)
	return mapCaptureSession(row, event.Title), true, nil
}

func (s *Store) AppendFenceCapturePoint(token string, point domain.FenceCapturePoint) (domain.FenceCaptureSession, error) {
	session, ok, err := s.GetFenceCaptureSessionByToken(token)
	if err != nil {
		return domain.FenceCaptureSession{}, err
	}
	if !ok {
		return domain.FenceCaptureSession{}, store.ErrCaptureNotFound
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		return domain.FenceCaptureSession{}, store.ErrCaptureExpired
	}

	points := session.Points
	if point.CapturedAt.IsZero() {
		point.CapturedAt = time.Now().UTC()
	}
	role := strings.TrimSpace(strings.ToLower(point.Role))
	point.Role = role

	if session.TargetShape == "circle" {
		switch role {
		case "center":
			filtered := make([]domain.FenceCapturePoint, 0, len(points))
			for _, p := range points {
				if p.Role != "center" {
					filtered = append(filtered, p)
				}
			}
			points = append(filtered, point)
		default:
			point.Role = "edge"
			points = append(points, point)
		}
	} else {
		point.Role = ""
		points = append(points, point)
	}

	pointsJSON, err := json.Marshal(points)
	if err != nil {
		return domain.FenceCaptureSession{}, err
	}
	if err := s.db.Model(&FenceCaptureSessionModel{}).
		Where("id = ?", session.ID).
		Update("points_json", string(pointsJSON)).Error; err != nil {
		return domain.FenceCaptureSession{}, err
	}
	session.Points = points
	return session, nil
}

func (s *Store) ApplyFenceCaptureSession(eventID, sessionID, name string, startAt, endAt time.Time) (domain.Fence, error) {
	var row FenceCaptureSessionModel
	if err := s.db.Where("id = ? AND event_id = ?", sessionID, eventID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.Fence{}, store.ErrCaptureNotFound
		}
		return domain.Fence{}, err
	}
	if row.Status != "active" {
		return domain.Fence{}, store.ErrCaptureNotFound
	}

	event, ok, err := s.GetEvent(eventID)
	if err != nil {
		return domain.Fence{}, err
	}
	if !ok {
		return domain.Fence{}, store.ErrEventNotFound
	}

	session := mapCaptureSession(row, event.Title)

	var fenceStart, fenceEnd *time.Time
	if !startAt.IsZero() {
		fenceStart = &startAt
	}
	if !endAt.IsZero() {
		fenceEnd = &endAt
	}
	resolvedStart, resolvedEnd, err := domain.ResolveFenceSchedule(event, fenceStart, fenceEnd)
	if err != nil {
		return domain.Fence{}, err
	}
	fenceName := strings.TrimSpace(name)
	if fenceName == "" {
		fenceName = domain.GenerateFenceName(event.Title, resolvedStart, resolvedEnd)
	}

	var fence domain.Fence
	switch session.TargetShape {
	case "circle":
		centerLat, centerLng, radiusM, err := domain.ResolveCircleFromCapturePoints(session.Points)
		if err != nil {
			return domain.Fence{}, store.ErrInvalidCapture
		}
		fence, err = s.AddFence(eventID, domain.FenceCreateInput{
			Name:      fenceName,
			ShapeType: "circle",
			StartAt:   resolvedStart,
			EndAt:     resolvedEnd,
			CenterLat: centerLat,
			CenterLng: centerLng,
			RadiusM:   radiusM,
		})
	default:
		if len(session.Points) < 3 {
			return domain.Fence{}, store.ErrInvalidCapture
		}
		pairs := make([][2]float64, 0, len(session.Points))
		for _, p := range session.Points {
			pairs = append(pairs, [2]float64{p.Lat, p.Lng})
		}
		polygonJSON, err := domain.BuildPolygonGeoJSON(pairs)
		if err != nil {
			return domain.Fence{}, err
		}
		fence, err = s.AddFence(eventID, domain.FenceCreateInput{
			Name:        fenceName,
			ShapeType:   "polygon",
			StartAt:     resolvedStart,
			EndAt:       resolvedEnd,
			PolygonJSON: polygonJSON,
		})
	}
	if err != nil {
		return domain.Fence{}, err
	}

	_ = s.db.Model(&row).Update("status", "applied").Error
	return fence, nil
}

func mapCaptureSession(row FenceCaptureSessionModel, eventTitle string) domain.FenceCaptureSession {
	points := []domain.FenceCapturePoint{}
	if strings.TrimSpace(row.PointsJSON) != "" {
		_ = json.Unmarshal([]byte(row.PointsJSON), &points)
	}
	shape := row.TargetShape
	if shape == "" {
		shape = "polygon"
	}
	return domain.FenceCaptureSession{
		ID:          row.ID,
		EventID:     row.EventID,
		EventTitle:  eventTitle,
		Token:       row.Token,
		TargetShape: shape,
		Status:      row.Status,
		Points:      points,
		ExpiresAt:   row.ExpiresAt.UTC(),
		CreatedAt:   row.CreatedAt.UTC(),
	}
}
