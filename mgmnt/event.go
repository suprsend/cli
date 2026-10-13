package mgmnt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/client"
)

type EventsResponse struct {
	Meta struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
	Results []any `json:"results"`
}

type ListEventsResponse struct {
	Results []Event `json:"results"`
	Meta    struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

type Event struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	PayloadSchema map[string]interface{} `json:"payload_schema"`
	CreatedAt     string                 `json:"created_at"`
}

func (c *SS_MgmntClient) ListEvents(workspace string, limit, offset int) (*ListEventsResponse, error) {
	client := client.NewHTTPClient()
	defer client.Close()

	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "event", "/")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed parsing url: %w", err)
	}
	q := u.Query()
	q.Add("limit", strconv.Itoa(limit))
	q.Add("offset", strconv.Itoa(offset))
	u.RawQuery = q.Encode()
	urlStr = u.String()
	log.Debugf("Getting Events for workspace: %s", workspace)
	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetResult(&ListEventsResponse{}).
		Get(urlStr)
	if err != nil {
		log.Errorf("Error getting events: %s", err)
		return nil, err
	}
	if res.IsError() {
		return nil, apiError(res)
	}
	events := res.Result().(*ListEventsResponse)
	return events, nil
}

func (c *SS_MgmntClient) GetEvents(workspace string) (*EventsResponse, error) {
	client := client.NewHTTPClient()
	defer client.Close()

	limit := 50
	offset := 0
	allEvents := []any{}
	totalCount := 0

	for {
		urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "event", "/")
		if err != nil {
			return nil, fmt.Errorf("failed constructing url: %w", err)
		}
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, fmt.Errorf("failed parsing url: %w", err)
		}
		q := u.Query()
		q.Add("limit", strconv.Itoa(limit))
		q.Add("offset", strconv.Itoa(offset))
		q.Add("has_linked_schema", "true")
		u.RawQuery = q.Encode()
		urlStr = u.String()
		res, err := client.R().
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetResult(&EventsResponse{}).
			Get(urlStr)
		if err != nil {
			log.Errorf("Error getting events: %s", err)
			return nil, err
		}
		if res.IsError() {
			return nil, apiError(res)
		}
		events := res.Result().(*EventsResponse)
		if len(events.Results) == 0 {
			break
		}
		allEvents = append(allEvents, events.Results...)
		totalCount += len(events.Results)
		offset += limit
	}
	return &EventsResponse{Results: allEvents}, nil
}

func (c *SS_MgmntClient) GetEventDetail(workspace, eventName string) (*Event, error) {
	client := client.NewHTTPClient()
	defer client.Close()

	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "event", eventName, "/")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	log.Debugf("Getting event detail for: %s", eventName)
	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetResult(&Event{}).
		Get(urlStr)
	if err != nil {
		return nil, err
	}
	if res.IsError() {
		return nil, apiError(res)
	}
	return res.Result().(*Event), nil
}

// EventPushRecord is the result for one event in a bulk push, in request order.
type EventPushRecord struct {
	StatusCode int    `json:"status_code"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Error      *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Failed reports whether the API rejected this event.
func (r EventPushRecord) Failed() bool {
	return r.Status != "success" || r.StatusCode >= 400
}

// PushEventsFromPayload upserts events with one bulk request and returns the
// per-event results. The endpoint answers 207 whenever the request itself is
// valid, even if some events fail (for example a payload_schema that doesn't
// exist in the workspace), so callers must check each record.
func (c *SS_MgmntClient) PushEventsFromPayload(workspace string, events map[string]any) ([]EventPushRecord, error) {
	client := client.NewHTTPClient()
	defer client.Close()

	urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", workspace, "bulk", "event", "/")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	log.Debugf("Pushing events to workspace: %s", workspace)
	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		SetBody(events).
		Post(urlStr)
	if err != nil {
		log.Errorf("Error pushing event: %s", err)
		return nil, err
	}
	if res.IsError() {
		return nil, apiError(res)
	}
	return parseEventPushRecords(res.Bytes()), nil
}

// parseEventPushRecords reads the bulk reply: {"records":[...], ...} when at
// least one event is valid, or a bare array of records when none is.
func parseEventPushRecords(body []byte) []EventPushRecord {
	body = bytes.TrimSpace(body)
	var records []EventPushRecord
	if len(body) > 0 && body[0] == '[' {
		if err := json.Unmarshal(body, &records); err != nil {
			log.Warnf("Couldn't parse event push response: %v", err)
		}
		return records
	}
	var wrapped struct {
		Records []EventPushRecord `json:"records"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &wrapped); err != nil {
			log.Warnf("Couldn't parse event push response: %v", err)
		}
	}
	return wrapped.Records
}
