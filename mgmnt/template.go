package mgmnt

import (
	"encoding/json"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/client"
)

type Template struct {
	Slug            string   `json:"slug"`
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	Tags            []string `json:"tags"`
	EnabledChannels []string `json:"enabled_channels"`
}

type TemplateAPIResponse struct {
	Results []Template `json:"results"`
	Meta    struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

func (c *SS_MgmntClient) ListTemplates(workspace string, limit int, offset int, mode string) (*TemplateAPIResponse, error) {
	if mode != "live" && mode != "draft" {
		return nil, fmt.Errorf("invalid mode: %s. Available modes are: live, draft", mode)
	}

	client := client.NewHTTPClient()
	defer client.Close()

	apiLimit := 50
	allTemplates := []Template{}
	currentOffset := offset
	remainingLimit := limit
	for remainingLimit > 0 {
		currentLimit := apiLimit
		if remainingLimit < apiLimit {
			currentLimit = remainingLimit
		}

		log.Debugf("Getting templates for workspace: %s, limit: %d, offset: %d", workspace, currentLimit, currentOffset)
		res, err := client.R().
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetResult(&TemplateAPIResponse{}).
			Get(c.mgmnt_base_URL + "v2/" + workspace + "/template/?limit=" + strconv.Itoa(currentLimit) + "&offset=" + strconv.Itoa(currentOffset) + "&mode=" + mode)
		if err != nil {
			log.Errorf("Error getting templates: %s", err)
			return nil, err
		}
		if res.IsError() {
			var errorResp ErrorResponse
			if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
				return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
			}
			return nil, fmt.Errorf("request failed: %s", res.Status())
		}

		templates := res.Result().(*TemplateAPIResponse)
		if len(templates.Results) == 0 {
			break
		}

		allTemplates = append(allTemplates, templates.Results...)
		remainingLimit -= len(templates.Results)
		currentOffset += len(templates.Results)
		if len(templates.Results) < currentLimit {
			break
		}
	}

	return &TemplateAPIResponse{
		Results: allTemplates,
		Meta: struct {
			Count  int `json:"count"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}{
			Count:  len(allTemplates),
			Limit:  limit,
			Offset: currentOffset,
		},
	}, nil
}
