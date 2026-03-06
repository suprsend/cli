package mgmnt

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/client"
)

type PreferenceTranslation struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PreferenceTranslationContent struct {
	Sections   map[string]PreferenceTranslation `json:"sections"`
	Categories map[string]PreferenceTranslation `json:"categories"`
}

type ListPreferenceTranslation struct {
	Results []struct {
		Locale string `json:"locale"`
	} `json:"results"`
}

func (c *SS_MgmntClient) ListPreferenceTranslations(workspace string) (*ListPreferenceTranslation, error) {
	client := client.NewHTTPClient()
	defer client.Close()

	url := fmt.Sprintf("%sv1/%s/preference_category/translation/locale", c.mgmnt_base_URL, workspace)
	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetResult(&ListPreferenceTranslation{}).
		Get(url)

	if err != nil {
		log.Errorf("Error getting translations: %s", err)
		return nil, err
	}
	if res.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
			return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("request failed with status: %s", res.Status())
	}

	translations := res.Result().(*ListPreferenceTranslation)

	return translations, nil
}

func (c *SS_MgmntClient) GetPreferenceTranslationsForLocale(workspace, locale string) (*PreferenceTranslationContent, error) {
	client := client.NewHTTPClient()
	defer client.Close()

	url := fmt.Sprintf("%sv1/%s/preference_category/translation/content/%s", c.mgmnt_base_URL, workspace, locale)
	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetResult(&PreferenceTranslationContent{}).
		Get(url)

	if err != nil {
		log.Errorf("Error getting preference translations: %s", err)
		return nil, err
	}
	if res.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
			return nil, fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("request failed with status: %s", res.Status())
	}

	translations := res.Result().(*PreferenceTranslationContent)

	return translations, nil
}

func (c *SS_MgmntClient) PushPreferenceTranslation(workspace, locale string, translation PreferenceTranslationContent) error {
	client := client.NewHTTPClient()
	defer client.Close()
	url := fmt.Sprintf("%sv1/%s/preference_category/translation/content/%s", c.mgmnt_base_URL, workspace, locale)
	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		SetBody(translation).
		Post(url)
	if err != nil {
		log.Errorf("Error pushing preference translations: %s", err)
		return err
	}
	if res.IsError() {
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(res.String()), &errorResp); err == nil {
			return fmt.Errorf("request failed with message: %s", errorResp.Message)
		}
		return fmt.Errorf("request failed with status: %s", res.Status())
	}
	return nil
}
