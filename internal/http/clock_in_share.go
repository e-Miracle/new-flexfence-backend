package http

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/flexfence/flexfence-backend/internal/domain"
)

func buildEventClockInShare(
	event domain.Event,
	qrToken string,
	issuedAt time.Time,
	joinPublicBase string,
) domain.EventClockInShare {
	share := domain.EventClockInShare{
		EventID:                 event.ID,
		EventTitle:              event.Title,
		ScanToClockInEnabled:    event.ScanToClockInEnabled,
		RotationIntervalMinutes: event.ClockInQRRotationMinutes,
	}
	if !event.ScanToClockInEnabled || strings.TrimSpace(qrToken) == "" {
		return share
	}

	issued := issuedAt.UTC()
	share.QRToken = qrToken
	share.IssuedAt = &issued
	if event.ClockInQRRotationMinutes > 0 {
		exp := issued.Add(time.Duration(event.ClockInQRRotationMinutes) * time.Minute)
		share.ExpiresAt = &exp
	}

	token := url.QueryEscape(qrToken)
	deep := fmt.Sprintf("flexfence://clock-in/%s?token=%s", event.ID, token)
	share.ClockInDeepLink = deep
	share.QRCodePayload = deep

	base := strings.TrimRight(strings.TrimSpace(joinPublicBase), "/")
	if base != "" {
		share.ClockInWebLink = fmt.Sprintf("%s/clock-in/%s?token=%s", base, event.ID, token)
	}

	return share
}
