package domain

import "errors"

// ResolveCircleFromCapturePoints derives a circle fence from captured GPS points.
// Circle capture expects one center point and at least one edge point (radius sample).
func ResolveCircleFromCapturePoints(points []FenceCapturePoint) (centerLat, centerLng, radiusM float64, err error) {
	if len(points) < 2 {
		return 0, 0, 0, errors.New("circle capture requires a center and at least one radius point")
	}

	var center *FenceCapturePoint
	edges := make([]FenceCapturePoint, 0, len(points))
	for i := range points {
		p := points[i]
		switch p.Role {
		case "center":
			center = &points[i]
		case "edge":
			edges = append(edges, p)
		default:
			edges = append(edges, p)
		}
	}

	if center == nil {
		center = &points[0]
		if len(points) == 1 {
			return 0, 0, 0, errors.New("circle capture requires a radius point")
		}
		if len(edges) == 0 || (edges[0].Lat == center.Lat && edges[0].Lng == center.Lng) {
			edges = points[1:]
		}
	}

	if len(edges) == 0 {
		return 0, 0, 0, errors.New("circle capture requires at least one radius point")
	}

	centerLat = center.Lat
	centerLng = center.Lng
	var totalRadius float64
	for _, edge := range edges {
		d := DistanceMeters(centerLat, centerLng, edge.Lat, edge.Lng)
		if d <= 0 {
			continue
		}
		totalRadius += d
	}
	if totalRadius <= 0 {
		return 0, 0, 0, errors.New("radius points must be away from the center")
	}
	radiusM = totalRadius / float64(len(edges))
	if radiusM < 5 {
		radiusM = 5
	}
	return centerLat, centerLng, radiusM, nil
}
