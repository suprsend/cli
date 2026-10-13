package mgmnt

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The bulk event endpoint always answers 207: an object with "records" when at
// least one event is valid, a bare array of records when none is.
func TestPushEventsFromPayloadRecords(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantFailed []bool
	}{
		{"all ok", `{"status":"success","total":1,"success":1,"failure":0,"records":[{"status_code":201,"status":"success","message":"ok"}]}`, []bool{false}},
		{"partial", `{"status":"success","total":2,"success":1,"failure":1,"records":[{"status_code":201,"status":"success"},{"status_code":400,"status":"error","message":"Schema 'x#3' not found or is not active.","error":{"message":"Schema 'x#3' not found or is not active.","type":"Not Found"}}]}`, []bool{false, true}},
		{"all failed", `[{"status_code":400,"status":"error","message":"Invalid event name","error":{"message":"Invalid event name","type":"ValidationError"}}]`, []bool{true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMultiStatus)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := NewClientWithUrls("token", srv.URL, srv.URL, false)

			records, err := c.PushEventsFromPayload("staging", map[string]any{"events": []any{}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(records) != len(tt.wantFailed) {
				t.Fatalf("got %d records, want %d", len(records), len(tt.wantFailed))
			}
			for i, want := range tt.wantFailed {
				if records[i].Failed() != want {
					t.Errorf("record %d: Failed() = %v, want %v", i, records[i].Failed(), want)
				}
			}
		})
	}
}
