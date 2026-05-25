package mgmnt

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type TranslationItem struct {
	Locale        string `json:"locale"`
	FileName      string `json:"filename"`
	VersionNo     *int   `json:"version_no"`
	VersionStatus string `json:"version_status"`
	Action        string `json:"action"`
	// Content is populated only when the caller passes
	// include_content=true. Translation payloads are free-form key/value
	// maps (the keys are template tokens, the values are the localized
	// strings), so map[string]any covers both flat and nested shapes
	// without losing information. omitempty keeps it out of the JSON
	// output when include_content was not requested.
	Content map[string]any `json:"content,omitempty"`
}

type ListTranslation struct {
	Results []TranslationItem `json:"results"`
	Meta    struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

type TranslationResponse struct {
	Results []any `json:"results"`
	Meta    struct {
		Count  int `json:"count"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

func (c *SS_MgmntClient) ListTranslations(ctx context.Context, workspace, mode, includeContent string, limit, offset int) (*ListTranslation, error) {
	client := c.restyClient()
	defer client.Close()

	url := fmt.Sprintf("%sv1/%s/translation/?mode=%s&limit=%d&offset=%d&include_content=%s&include_version_info=true", c.mgmnt_base_URL, workspace, mode, limit, offset, includeContent)
	res, err := client.R().
		SetContext(ctx).
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetResult(&ListTranslation{}).
		Get(url)

	if err != nil {
		log.Errorf("Error getting translations: %s", err)
		return nil, err
	}
	if res.IsError() {
		return nil, apiError(res)
	}

	translations := res.Result().(*ListTranslation)

	return translations, nil
}

func (c *SS_MgmntClient) GetTranslations(ctx context.Context, workspace, mode string) (*TranslationResponse, error) {
	if mode != "live" && mode != "draft" {
		log.Errorf("%s: invalid mode. Available modes are: draft, live", mode)
		return nil, nil
	}
	client := c.restyClient()
	defer client.Close()

	limit := 10
	offset := 0
	allTranslations := []any{}
	totalCount := 0

	for {
		res, err := client.R().
			SetContext(ctx).
			SetDebug(c.debug).
			SetHeader("Authorization", "ServiceToken "+c.serviceToken).
			SetResult(&TranslationResponse{}).
			Get(c.mgmnt_base_URL + "v1/" + workspace + "/translation/?include_content=true&include_version_info=true" + "&limit=" + strconv.Itoa(limit) + "&offset=" + strconv.Itoa(offset) + "&mode=" + mode)
		if err != nil {
			log.Errorf("Failed to get translations: %v", err)
			return nil, err
		}

		if res.IsError() {
			return nil, apiError(res)
		}

		translations := res.Result().(*TranslationResponse)

		if len(translations.Results) == 0 {
			break
		}

		allTranslations = append(allTranslations, translations.Results...)
		totalCount += len(translations.Results)
		offset += limit
	}

	return &TranslationResponse{
		Results: allTranslations,
		Meta: struct {
			Count  int `json:"count"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}{
			Count:  totalCount,
			Limit:  limit,
			Offset: 0,
		},
	}, nil
}

func (c *SS_MgmntClient) PushTranslation(ctx context.Context, workspace, filename string, translation map[string]any) error {
	client := c.restyClient()
	defer client.Close()
	url := fmt.Sprintf("%sv1/%s/translation/content/%s/", c.mgmnt_base_URL, workspace, filename)
	res, err := client.R().
		SetContext(ctx).
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetBody(translation).
		Post(url)
	if err != nil {
		log.Errorf("Error pushing translations: %s", err)
		return err
	}
	if res.IsError() {
		return apiError(res)
	}
	return nil
}

func (c *SS_MgmntClient) FinalizeTranslation(ctx context.Context, workspace, commitMessage string) error {
	client := c.restyClient()
	defer client.Close()
	encodedCommitMessage := url.QueryEscape(commitMessage)
	url := fmt.Sprintf("%sv1/%s/translation/commit/?commit_message=%s", c.mgmnt_base_URL, workspace, encodedCommitMessage)
	res, err := client.R().
		SetContext(ctx).
		SetDebug(c.debug).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		Patch(url)
	if err != nil {
		return err
	}
	if res.IsError() {
		return apiError(res)
	}
	return nil
}
