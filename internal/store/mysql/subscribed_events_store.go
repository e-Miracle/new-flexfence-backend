package mysql

import (
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
	"strings"
)

func (s *Store) ListSubscribedGeofenceEvents(userID string, now time.Time) ([]domain.SubscribedGeofenceEvent, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	page, err := s.ListJoinsByUserFiltered(userID, store.UserEventJoinFilter{Page: 1, Limit: 500})
	if err != nil {
		return nil, err
	}

	out := make([]domain.SubscribedGeofenceEvent, 0, len(page.Joins))
	for _, join := range page.Joins {
		if !join.EventEndAt.IsZero() && join.EventEndAt.Before(now) {
			continue
		}
		fences, err := s.ListFencesByEvent(join.EventID)
		if err != nil {
			return nil, err
		}
		circleFences := make([]domain.Fence, 0, len(fences))
		for _, f := range fences {
			switch f.ShapeType {
			case "circle":
				if f.RadiusM <= 0 || (f.CenterLat == 0 && f.CenterLng == 0) {
					continue
				}
				circleFences = append(circleFences, f)
			case "polygon":
				if strings.TrimSpace(f.PolygonJSON) == "" {
					continue
				}
				circleFences = append(circleFences, f)
			}
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
