package domain

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
)

type geoJSONPolygon struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

// ParsePolygonRing extracts the outer ring as [lat, lng] pairs from stored GeoJSON.
func ParsePolygonRing(polygonJSON string) ([][]float64, error) {
	raw := strings.TrimSpace(polygonJSON)
	if raw == "" {
		return nil, errors.New("polygon_geojson is required")
	}
	var poly geoJSONPolygon
	if err := json.Unmarshal([]byte(raw), &poly); err != nil {
		return nil, err
	}
	if !strings.EqualFold(poly.Type, "Polygon") || len(poly.Coordinates) == 0 {
		return nil, errors.New("polygon_geojson must be a GeoJSON Polygon")
	}
	ring := poly.Coordinates[0]
	if len(ring) < 4 {
		return nil, errors.New("polygon must have at least 3 points")
	}
	out := make([][]float64, 0, len(ring))
	for _, coord := range ring {
		if len(coord) < 2 {
			continue
		}
		out = append(out, []float64{coord[1], coord[0]})
	}
	if len(out) < 3 {
		return nil, errors.New("polygon must have at least 3 points")
	}
	return out, nil
}

// BuildPolygonGeoJSON builds a closed GeoJSON Polygon from [lat, lng] pairs.
func BuildPolygonGeoJSON(points [][2]float64) (string, error) {
	if len(points) < 3 {
		return "", errors.New("at least 3 points are required")
	}
	ring := make([][]float64, 0, len(points)+1)
	for _, p := range points {
		ring = append(ring, []float64{p[1], p[0]})
	}
	first := ring[0]
	last := ring[len(ring)-1]
	if first[0] != last[0] || first[1] != last[1] {
		ring = append(ring, []float64{first[0], first[1]})
	}
	payload := geoJSONPolygon{
		Type:        "Polygon",
		Coordinates: [][][]float64{ring},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// PointInPolygonRing uses ray casting on a [lat,lng] ring.
func PointInPolygonRing(ring [][]float64, lat, lng float64) bool {
	if len(ring) < 3 {
		return false
	}
	inside := false
	j := len(ring) - 1
	for i := 0; i < len(ring); i++ {
		latI, lngI := ring[i][0], ring[i][1]
		latJ, lngJ := ring[j][0], ring[j][1]
		intersects := (lngI > lng) != (lngJ > lng) &&
			lat < (latJ-latI)*(lng-lngI)/(lngJ-lngI+0.0)+latI
		if intersects {
			inside = !inside
		}
		j = i
	}
	return inside
}

// PointInPolygonFence reports whether a point lies inside a polygon fence.
func PointInPolygonFence(f Fence, lat, lng float64) bool {
	if f.ShapeType != "polygon" || strings.TrimSpace(f.PolygonJSON) == "" {
		return false
	}
	if lat == 0 && lng == 0 {
		return false
	}
	ring, err := ParsePolygonRing(f.PolygonJSON)
	if err != nil {
		return false
	}
	return PointInPolygonRing(ring, lat, lng)
}

// PointInPolygonFenceWithAccuracy allows fixes near the polygon boundary within bufferM.
func PointInPolygonFenceWithAccuracy(f Fence, lat, lng, bufferM float64) bool {
	if f.ShapeType != "polygon" || strings.TrimSpace(f.PolygonJSON) == "" {
		return false
	}
	if lat == 0 && lng == 0 {
		return false
	}
	ring, err := ParsePolygonRing(f.PolygonJSON)
	if err != nil {
		return false
	}
	if PointInPolygonRing(ring, lat, lng) {
		return true
	}
	return DistanceToPolygonRingM(ring, lat, lng) <= bufferM
}

// DistanceToPolygonRingM returns the shortest distance in meters from a point to the polygon boundary.
func DistanceToPolygonRingM(ring [][]float64, lat, lng float64) float64 {
	if len(ring) < 2 {
		return math.MaxFloat64
	}
	minDist := math.MaxFloat64
	for i := 0; i < len(ring); i++ {
		j := (i + 1) % len(ring)
		d := DistanceToSegmentM(ring[i][0], ring[i][1], ring[j][0], ring[j][1], lat, lng)
		if d < minDist {
			minDist = d
		}
	}
	return minDist
}

// DistanceToSegmentM approximates the shortest distance from a point to a segment in meters.
func DistanceToSegmentM(latA, lngA, latB, lngB, latP, lngP float64) float64 {
	dLat := latB - latA
	dLng := lngB - lngA
	if dLat == 0 && dLng == 0 {
		return DistanceMeters(latA, lngA, latP, lngP)
	}
	t := ((latP-latA)*dLat + (lngP-lngA)*dLng) / (dLat*dLat + dLng*dLng)
	if t < 0 {
		return DistanceMeters(latA, lngA, latP, lngP)
	}
	if t > 1 {
		return DistanceMeters(latB, lngB, latP, lngP)
	}
	closestLat := latA + t*dLat
	closestLng := lngA + t*dLng
	return DistanceMeters(closestLat, closestLng, latP, lngP)
}
