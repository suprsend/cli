package event

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
)

type EventWriteStats struct {
	Total   int
	Success int
	Failed  int
	Errors  []string
}

var unsafeCharsRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)
var leadingNonAlphanumRe = regexp.MustCompile(`^[^a-zA-Z0-9]+`)
var trailingNonAlphanumRe = regexp.MustCompile(`[^a-zA-Z0-9]+$`)

func promptForOutputDirectory() (string, bool) {
	if !utils.IsInputInteractive() {
		fmt.Fprintf(os.Stderr, "required flag missing, cannot prompt in non-interactive mode")
		return "", false
	}
	reader := bufio.NewReader(os.Stdin)
	defaultDir := filepath.Join(".", "suprsend", "events")
	fmt.Fprintf(os.Stdout, "Where would you like to save the events?\n")
	fmt.Fprintf(os.Stdout, "Default: %s\n", defaultDir)
	fmt.Fprintf(os.Stdout, "Enter directory path (or press Enter for default): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v. Using default directory: %s\n", err, defaultDir)
		return defaultDir, true
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultDir, true
	}
	return input, true
}

// slugifyEventName converts an event name to a filesystem-safe directory name.
// A leading $ is preserved (private event convention). Other unsafe chars become _.
// Returns an error if the name has no representable form.
func slugifyEventName(name string) (string, error) {
	prefix := ""
	body := name
	if strings.HasPrefix(name, "$") {
		prefix = "$"
		body = name[1:]
	}

	body = unsafeCharsRe.ReplaceAllString(body, "_")
	body = leadingNonAlphanumRe.ReplaceAllString(body, "")
	body = trailingNonAlphanumRe.ReplaceAllString(body, "")

	// Truncate so total length (prefix + body) <= 128
	maxBody := 128 - len(prefix)
	if len(body) > maxBody {
		body = body[:maxBody]
		body = trailingNonAlphanumRe.ReplaceAllString(body, "")
	}

	if body == "" {
		return "", fmt.Errorf("event name %q has no representable filesystem form — rename it in the SuprSend UI", name)
	}
	return prefix + body, nil
}

func WriteEventsToFiles(events_resp *mgmnt.EventsResponse, outputDir string) (*EventWriteStats, error) {
	stats := &EventWriteStats{
		Total:  len(events_resp.Results),
		Errors: []string{},
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return stats, fmt.Errorf("failed to create output directory: %w", err)
	}

	// dirName → original event name, for collision detection
	seen := map[string]string{}

	for _, result := range events_resp.Results {
		obj, ok := result.(map[string]any)
		if !ok {
			stats.Failed++
			stats.Errors = append(stats.Errors, "invalid event format (not a JSON object)")
			continue
		}

		name, _ := obj["name"].(string)
		if name == "" {
			stats.Failed++
			stats.Errors = append(stats.Errors, "event missing name field")
			continue
		}

		dirName, err := slugifyEventName(name)
		if err != nil {
			stats.Failed++
			stats.Errors = append(stats.Errors, err.Error())
			log.Warnf("events: skip %q: %v", name, err)
			continue
		}

		if existing, collision := seen[strings.ToLower(dirName)]; collision {
			stats.Failed++
			msg := fmt.Sprintf(
				"events: names %q and %q collide on directory %q; rename one in the SuprSend UI and re-pull",
				existing, name, dirName,
			)
			stats.Errors = append(stats.Errors, msg)
			log.Warnf("Skipping %q: %s", name, msg)
			continue
		}
		seen[strings.ToLower(dirName)] = name

		eventDir := filepath.Join(outputDir, dirName)
		if err := os.MkdirAll(eventDir, 0o755); err != nil {
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("failed to create directory for %q: %v", name, err))
			continue
		}

		fileData, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("failed to marshal event %q: %v", name, err))
			log.Errorf("Failed to marshal event %q: %v", name, err)
			continue
		}

		filename := filepath.Join(eventDir, "event.json")
		if err := os.WriteFile(filename, append(fileData, '\n'), 0o644); err != nil {
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("failed to write %s: %v", filename, err))
			log.Errorf("Failed to write %s: %v", filename, err)
			continue
		}

		log.Infof("Wrote event to %s", filename)
		stats.Success++
	}

	return stats, nil
}

// ReadEventsFromDir reads per-event subdirectories and returns the events slice
// ready for the push wire format. $schema is stripped before returning.
func ReadEventsFromDir(dirPath string) ([]any, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read events directory %q: %w", dirPath, err)
	}

	var events []any
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		eventFile := filepath.Join(dirPath, entry.Name(), "event.json")
		data, err := os.ReadFile(eventFile)
		if err != nil {
			return nil, fmt.Errorf("events/%s: missing event.json: %w", entry.Name(), err)
		}
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("events/%s/event.json: invalid JSON: %w", entry.Name(), err)
		}
		delete(obj, "$schema")
		events = append(events, obj)
	}
	return events, nil
}
