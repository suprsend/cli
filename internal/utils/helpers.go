package utils

import (
	"encoding/json"
)

// DeepCopyMap returns a deep copy of a map[string]any via JSON round-trip.
func DeepCopyMap(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	json.Unmarshal(b, &out)
	return out
}

func GetExtensionForLanguage(language string) string {
	extMap := map[string]string{
		"python":         ".py",
		"java":           ".java",
		"typescript":     ".ts",
		"typescript-zod": ".ts",
		"go":             ".go",
		"kotlin":         ".kt",
		"swift":          ".swift",
		"dart":           ".dart",
	}
	return extMap[language]
}

func Remarshal(src any, dst any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
