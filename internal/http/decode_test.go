package http

import (
	"net/http"
	"strings"
	"testing"
)

func TestDecodeCreateEventRequest(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "dashboard payload",
			body: `{"title":"Tech Conference","description":"","start_at":"2026-05-28T08:00:00.000Z","end_at":"2026-05-28T16:00:00.000Z"}`,
		},
		{
			name: "readme example without schedule",
			body: `{"title":"Tech Conference","description":"Main hall event"}`,
		},
		{
			name:    "camelCase unknown fields",
			body:    `{"title":"x","startAt":"2026-05-28T08:00:00.000Z","endAt":"2026-05-28T16:00:00.000Z"}`,
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    ``,
			wantErr: true,
		},
		{
			name:    "invalid syntax",
			body:    `{title:"x"}`,
			wantErr: true,
		},
		{
			name: "null datetime fields decodes as empty strings",
			body: `{"title":"x","start_at":null,"end_at":"2026-05-28T16:00:00.000Z"}`,
		},
		{
			name:    "title as number",
			body:    `{"title":123,"start_at":"2026-05-28T08:00:00.000Z","end_at":"2026-05-28T16:00:00.000Z"}`,
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var req CreateEventRequest
			r, _ := http.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(tc.body))
			err := decodeJSON(r, &req)
			if (err != nil) != tc.wantErr {
				t.Fatalf("decode err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
