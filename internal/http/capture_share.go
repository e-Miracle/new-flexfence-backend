package http

import (
	"fmt"
	"net/url"
	"strings"
)

func buildFenceCaptureLink(eventID, token, joinPublicBase string) string {
	encoded := url.QueryEscape(token)
	deep := fmt.Sprintf("flexfence://capture/%s?token=%s", eventID, encoded)
	base := strings.TrimRight(strings.TrimSpace(joinPublicBase), "/")
	if base != "" {
		return fmt.Sprintf("%s/capture/%s?token=%s", base, eventID, encoded)
	}
	return deep
}

func buildFenceCaptureDeepLink(eventID, token string) string {
	return fmt.Sprintf("flexfence://capture/%s?token=%s", eventID, url.QueryEscape(token))
}
