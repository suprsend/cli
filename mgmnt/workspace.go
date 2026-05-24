package mgmnt

import (
	"fmt"
	"net/url"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Workspace struct {
	UID             string `json:"uid"`
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Mode            string `json:"mode"`
	IsExpired       bool   `json:"is_expired"`
	HasLimitReached bool   `json:"has_limit_reached"`
}

type WorkspaceListResponse struct {
	Results []Workspace `json:"results"`
	Meta    struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

func (c *SS_MgmntClient) ListWorkspaces(limit, offset int) (*WorkspaceListResponse, error) {
	httpClient := c.restyClient()
	defer httpClient.Close()

	apiLimit := 50
	allWorkspaces := []Workspace{}
	currentOffset := offset
	remainingLimit := limit

	for remainingLimit > 0 {
		currentLimit := min(apiLimit, remainingLimit)

		log.Debugf("Getting workspaces: limit=%d, offset=%d", currentLimit, currentOffset)

		urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", "workspace", "/")
		if err != nil {
			return nil, fmt.Errorf("failed constructing url: %w", err)
		}
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, fmt.Errorf("failed parsing url: %w", err)
		}
		q := u.Query()
		q.Add("limit", strconv.Itoa(currentLimit))
		q.Add("offset", strconv.Itoa(currentOffset))
		u.RawQuery = q.Encode()
		urlStr = u.String()

		res, err := httpClient.R().
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetResult(&WorkspaceListResponse{}).
			Get(urlStr)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		if res.IsError() {
			return nil, apiError(res)
		}

		page := res.Result().(*WorkspaceListResponse)
		if len(page.Results) == 0 {
			break
		}

		allWorkspaces = append(allWorkspaces, page.Results...)
		remainingLimit -= len(page.Results)
		currentOffset += len(page.Results)
		if len(page.Results) < currentLimit {
			break
		}
	}

	return &WorkspaceListResponse{
		Results: allWorkspaces,
		Meta: struct {
			Count  int `json:"count"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}{
			Count:  len(allWorkspaces),
			Limit:  limit,
			Offset: currentOffset,
		},
	}, nil
}
