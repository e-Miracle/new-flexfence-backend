package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/flexfence/flexfence-backend/internal/domain"
	"github.com/flexfence/flexfence-backend/internal/store"
)

const defaultFenceShapeType = "circle"

func deleteFenceHandler(dataStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID, fenceID := fenceIDsFromPath(r.URL.Path)
		if eventID == "" || fenceID == "" {
			writeAPIError(w, http.StatusBadRequest, "fence_id_required", "Fence id is required")
			return
		}
		if err := dataStore.DeleteFence(eventID, fenceID); err != nil {
			switch {
			case errors.Is(err, store.ErrEventNotFound):
				writeAPIError(w, http.StatusNotFound, "event_not_found", "Event was not found")
			case errors.Is(err, store.ErrFenceNotFound):
				writeAPIError(w, http.StatusNotFound, "fence_not_found", "Fence was not found")
			case errors.Is(err, store.ErrEventLive):
				writeAPIError(w, http.StatusConflict, "event_live", "Fences cannot be deleted while the event is live")
			default:
				writeAPIError(w, http.StatusInternalServerError, "internal_error", "An internal server error occurred")
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func fenceIDsFromPath(path string) (eventID, fenceID string) {
	path = strings.TrimPrefix(path, "/v1/events/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[1] != "fences" {
		return "", ""
	}
	return parts[0], parts[2]
}

func resolveCreateFenceInput(event domain.Event, req CreateFenceRequest) (domain.FenceCreateInput, error) {
	shapeType := strings.TrimSpace(strings.ToLower(req.ShapeType))
	if shapeType == "" {
		shapeType = defaultFenceShapeType
	}
	if shapeType != "circle" && shapeType != "polygon" {
		return domain.FenceCreateInput{}, errors.New("shape_type must be circle or polygon")
	}
	if shapeType == "polygon" && strings.TrimSpace(req.PolygonJSON) == "" {
		return domain.FenceCreateInput{}, errors.New("polygon_geojson is required for polygon fences")
	}
	if shapeType == "polygon" {
		if _, err := domain.ParsePolygonRing(req.PolygonJSON); err != nil {
			return domain.FenceCreateInput{}, err
		}
	}

	fenceStart, err := parseOptionalRFC3339Time(req.StartAt, "start_at")
	if err != nil {
		return domain.FenceCreateInput{}, err
	}
	fenceEnd, err := parseOptionalRFC3339Time(req.EndAt, "end_at")
	if err != nil {
		return domain.FenceCreateInput{}, err
	}

	startAt, endAt, err := domain.ResolveFenceSchedule(event, fenceStart, fenceEnd)
	if err != nil {
		return domain.FenceCreateInput{}, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = domain.GenerateFenceName(event.Title, startAt, endAt)
	}

	return domain.FenceCreateInput{
		Name:        name,
		ShapeType:   shapeType,
		StartAt:     startAt,
		EndAt:       endAt,
		CenterLat:   req.CenterLat,
		CenterLng:   req.CenterLng,
		RadiusM:     req.RadiusM,
		PolygonJSON: req.PolygonJSON,
	}, nil
}
