package mgmnt

import (
	"fmt"
	"strings"

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
		return nil, apiError(res)
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
		return nil, apiError(res)
	}

	translations := res.Result().(*PreferenceTranslationContent)

	return translations, nil
}

// SkippedTranslationKeys are the section and category keys a translation
// push left out: keys that aren't slugs, and keys for sections or categories
// the workspace doesn't have. Older versions of the service don't report
// them.
type SkippedTranslationKeys struct {
	Sections   []string `json:"sections"`
	Categories []string `json:"categories"`
}

// Note describes the skipped keys, or returns "" when there are none.
func (s SkippedTranslationKeys) Note() string {
	var parts []string
	if len(s.Sections) > 0 {
		parts = append(parts, "sections "+strings.Join(s.Sections, ", "))
	}
	if len(s.Categories) > 0 {
		parts = append(parts, "categories "+strings.Join(s.Categories, ", "))
	}
	if len(parts) == 0 {
		return ""
	}
	return "not saved (not a section or category slug in the workspace): " + strings.Join(parts, "; ")
}

// PushPreferenceTranslation saves one locale's translations and returns the
// keys the API skipped.
func (c *SS_MgmntClient) PushPreferenceTranslation(workspace, locale string, translation PreferenceTranslationContent) (SkippedTranslationKeys, error) {
	var reply struct {
		Skipped SkippedTranslationKeys `json:"skipped"`
	}
	client := client.NewHTTPClient()
	defer client.Close()
	url := fmt.Sprintf("%sv1/%s/preference_category/translation/content/%s", c.mgmnt_base_URL, workspace, locale)
	res, err := client.R().
		SetDebug(c.debug).
		SetHeader("Authorization", "ServiceToken "+c.serviceToken).
		SetHeader("Content-Type", "application/json").
		SetBody(translation).
		SetResult(&reply).
		Post(url)
	if err != nil {
		log.Errorf("Error pushing preference translations: %s", err)
		return SkippedTranslationKeys{}, err
	}
	if res.IsError() {
		return SkippedTranslationKeys{}, apiError(res)
	}
	return reply.Skipped, nil
}
