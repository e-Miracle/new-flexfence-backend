package mysql

import (
	"errors"
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
	"gorm.io/gorm"
)

func (s *Store) GetFenceByEvent(eventID, fenceID string) (domain.Fence, bool, error) {
	var m FenceModel
	err := s.db.Where("id = ? AND event_id = ?", fenceID, eventID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Fence{}, false, nil
	}
	if err != nil {
		return domain.Fence{}, false, err
	}
	return mapFenceModel(m), true, nil
}

func (s *Store) ListJoinsByEvent(eventID string) ([]domain.EventJoin, error) {
	var rows []EventJoinModel
	if err := s.db.Where("event_id = ?", eventID).Order("joined_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.EventJoin, 0, len(rows))
	for _, r := range rows {
		out = append(out, mapJoinModel(r))
	}
	return out, nil
}

func (s *Store) ListJoinsByUser(userID string) ([]domain.UserEventJoin, error) {
	page, err := s.ListJoinsByUserFiltered(userID, store.UserEventJoinFilter{Page: 1, Limit: 1000})
	if err != nil {
		return nil, err
	}
	return page.Joins, nil
}

type userJoinRow struct {
	ID               string    `gorm:"column:id"`
	EventID          string    `gorm:"column:event_id"`
	JoinSource       string    `gorm:"column:join_source"`
	JoinedAt         time.Time `gorm:"column:joined_at"`
	EventTitle       string    `gorm:"column:event_title"`
	EventDescription string    `gorm:"column:event_description"`
	EventStartAt     time.Time `gorm:"column:event_start_at"`
	EventEndAt       time.Time `gorm:"column:event_end_at"`
	EventStatus      string    `gorm:"column:event_status"`
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

	base := s.db.Table("event_joins ej").
		Joins("JOIN events e ON e.id = ej.event_id").
		Where("ej.user_id = ?", userID)

	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		base = base.Where("(e.title LIKE ? OR e.description LIKE ?)", like, like)
	}
	if filter.Status != "" {
		base = base.Where("e.status = ?", filter.Status)
	}
	if filter.JoinSource != "" {
		base = base.Where("ej.join_source = ?", filter.JoinSource)
	}
	if filter.StartFrom != nil {
		base = base.Where("e.start_at >= ?", *filter.StartFrom)
	}
	if filter.StartTo != nil {
		base = base.Where("e.start_at <= ?", *filter.StartTo)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return store.UserEventJoinPage{}, err
	}

	offset := (filter.Page - 1) * filter.Limit
	var rows []userJoinRow
	err := base.Select(`
			ej.id,
			ej.event_id,
			ej.join_source,
			ej.joined_at,
			e.title AS event_title,
			e.description AS event_description,
			e.start_at AS event_start_at,
			e.end_at AS event_end_at,
			e.status AS event_status
		`).
		Order("e.start_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Scan(&rows).Error
	if err != nil {
		return store.UserEventJoinPage{}, err
	}

	joins := make([]domain.UserEventJoin, 0, len(rows))
	for _, r := range rows {
		joins = append(joins, domain.UserEventJoin{
			ID:               r.ID,
			EventID:          r.EventID,
			EventTitle:       r.EventTitle,
			EventDescription: r.EventDescription,
			EventStartAt:     r.EventStartAt,
			EventEndAt:       r.EventEndAt,
			EventStatus:      r.EventStatus,
			JoinSource:       r.JoinSource,
			JoinedAt:         r.JoinedAt,
		})
	}

	hasMore := int64(filter.Page*filter.Limit) < total
	return store.UserEventJoinPage{
		Joins:   joins,
		Total:   int(total),
		Page:    filter.Page,
		Limit:   filter.Limit,
		HasMore: hasMore,
	}, nil
}

func (s *Store) DeleteUserEventJoin(userID, joinID string) error {
	res := s.db.Where("id = ? AND user_id = ?", joinID, userID).Delete(&EventJoinModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return store.ErrJoinNotFound
	}
	return nil
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
		Event:      event,
		Fences:     fences,
		Joins:      joins,
		Attendance: attendance,
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
	fences, err := s.ListFencesByEvent(eventID)
	if err != nil {
		return domain.FenceAnalytics{}, err
	}
	attendance, err := s.ListAttendanceByEvent(eventID)
	if err != nil {
		return domain.FenceAnalytics{}, err
	}
	return domain.ComputeFenceAnalytics(fence, event, fences, attendance), nil
}
