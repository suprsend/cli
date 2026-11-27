package mgmnt

import (
	"fmt"
	"net/url"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/client"
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

// ListWorkspaces lists the workspaces the token can reach, starting at
// offset: limit of them, or all of them when limit <= 0. Meta.Count is the
// total the API reports.
func (c *SS_MgmntClient) ListWorkspaces(limit, offset int) (*WorkspaceListResponse, error) {
	httpClient := client.NewHTTPClient()
	defer httpClient.Close()

	all, total, err := listPages(limit, offset, func(limit, offset int) ([]Workspace, int, error) {
		log.Debugf("Getting workspaces: limit=%d, offset=%d", limit, offset)

		urlStr, err := url.JoinPath(c.mgmnt_base_URL, "v1", "workspace", "/")
		if err != nil {
			return nil, 0, fmt.Errorf("failed constructing url: %w", err)
		}
		u, err := url.Parse(urlStr)
		if err != nil {
			return nil, 0, fmt.Errorf("failed parsing url: %w", err)
		}
		q := u.Query()
		q.Add("limit", strconv.Itoa(limit))
		q.Add("offset", strconv.Itoa(offset))
		u.RawQuery = q.Encode()

		res, err := httpClient.R().
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetResult(&WorkspaceListResponse{}).
			Get(u.String())
		if err != nil {
			return nil, 0, fmt.Errorf("request failed: %w", err)
		}
		if res.IsError() {
			return nil, 0, apiError(res)
		}
		page := res.Result().(*WorkspaceListResponse)
		return page.Results, page.Meta.Count, nil
	})
	if err != nil {
		return nil, err
	}

	resp := &WorkspaceListResponse{Results: all}
	resp.Meta.Count = listTotal(total, offset, len(all))
	resp.Meta.Limit = limit
	resp.Meta.Offset = offset
	return resp, nil
}
