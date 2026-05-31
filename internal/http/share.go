package http

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/flexfence/flexfence-backend/internal/domain"
)

func buildEventShare(event domain.Event, qrToken, joinPublicBase string) domain.EventShare {
	token := url.QueryEscape(qrToken)
	deep := fmt.Sprintf("flexfence://join/%s?token=%s", event.ID, token)

	share := domain.EventShare{
		EventID:       event.ID,
		EventTitle:    event.Title,
		QRToken:       qrToken,
		JoinDeepLink:  deep,
		QRCodePayload: deep,
	}

	base := strings.TrimRight(strings.TrimSpace(joinPublicBase), "/")
	if base != "" {
		web := fmt.Sprintf("%s/join/%s?token=%s", base, event.ID, token)
		share.JoinWebLink = web
	}

	return share
}
