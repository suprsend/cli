// Package state keeps .suprsend-state.json, the record that lets push tell
// what changed since the last pull or push, and where.
//
// The file lives in an asset directory (next to the workflow or template
// directories it describes) and holds, for each workspace and item, two
// fingerprints taken the last time the CLI pulled or pushed that item: one of
// the item as the workspace had it, one of the local copy. On the next push,
// comparing both with the current values says whether the item changed
// locally, in the workspace, or both. Fingerprints are kept per API host and
// workspace, so files pulled from one workspace and pushed to another (or to
// a workspace of the same name on another host) don't read as a conflict. The file is meant to be committed with the asset files, like a
// lockfile.
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/suprsend/cli/internal/clierr"
)

// FileName is the state file's name inside an asset directory.
const FileName = ".suprsend-state.json"

// version 2 keys entries by API host as well as workspace. Version 1 had
// workspaces at the top level; it only shipped in QA builds.
const version = 2

// Entry holds an item's fingerprints from the last pull or push.
type Entry struct {
	Remote string `json:"remote"` // the workspace's draft of the item
	Local  string `json:"local"`  // the local copy
	// Pending is set by a pull of the live version when the workspace's
	// draft differed from it: the draft holds changes the local copy never
	// had. Push leaves such a draft alone until the local copy changes.
	Pending bool `json:"pending,omitempty"`
}

// File is a loaded state file. Its methods are safe for concurrent use.
type File struct {
	Version int `json:"version"`
	// Hosts maps API host > workspace > item key to its entry. Every account
	// has a "staging" and a "production" workspace, so the workspace name
	// alone doesn't say which one an entry describes.
	Hosts map[string]map[string]map[string]Entry `json:"hosts"`
	// Workspaces is the version 1 layout, read only to migrate it.
	Workspaces map[string]map[string]Entry `json:"workspaces,omitempty"`

	host  string
	path  string
	dirty bool
	mu    sync.Mutex
}

// Load reads dir's state file for the API at apiURL (the management URL in
// use). A missing file is an empty state: push then has no record to compare
// with and pushes what differs from the workspace.
func Load(dir, apiURL string) (*File, error) {
	f := &File{Version: version, Hosts: map[string]map[string]map[string]Entry{}, host: hostKey(apiURL), path: filepath.Join(dir, FileName)}
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		return f, nil
	}
	if err != nil {
		return nil, clierr.Wrap(err, clierr.CodeFileNotFound, "")
	}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, clierr.New(fmt.Sprintf("%s is not valid JSON: %v", f.path, err), clierr.CodeFileParseFailed).
			WithHint("fix it, or delete it: the next pull writes a new one, and until then push can't tell what changed in the workspace since your last pull")
	}
	if f.Version > version {
		return nil, clierr.New(fmt.Sprintf("%s was written by a newer version of the CLI (state version %d)", f.path, f.Version), clierr.CodeFileParseFailed).
			WithHint("upgrade the CLI")
	}
	if f.Hosts == nil {
		f.Hosts = map[string]map[string]map[string]Entry{}
	}
	if f.Version < version || f.Workspaces != nil {
		// Version 1 didn't record the host. Its entries are taken to be for
		// the host in use, which is where they almost always came from.
		for ws, entries := range f.Workspaces {
			if _, ok := f.Hosts[f.host][ws]; !ok && len(entries) > 0 {
				f.entries(ws, true)
				f.Hosts[f.host][ws] = entries
			}
		}
		f.Workspaces = nil
		f.dirty = true
	}
	f.Version = version
	return f, nil
}

// hostKey reduces an API URL to its host, lower-cased, so the same API
// written with or without a scheme or trailing slash gives one key.
func hostKey(apiURL string) string {
	raw := strings.TrimSpace(apiURL)
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return strings.ToLower(u.Host)
	}
	return strings.ToLower(strings.Trim(raw, "/"))
}

// entries returns the entries of workspace on the host in use, creating the
// maps when create is set. The caller holds f.mu.
func (f *File) entries(workspace string, create bool) map[string]Entry {
	if create {
		if f.Hosts[f.host] == nil {
			f.Hosts[f.host] = map[string]map[string]Entry{}
		}
		if f.Hosts[f.host][workspace] == nil {
			f.Hosts[f.host][workspace] = map[string]Entry{}
		}
	}
	return f.Hosts[f.host][workspace]
}

// Get returns the entry for key in workspace. A nil File has no entries.
func (f *File) Get(workspace, key string) (Entry, bool) {
	if f == nil {
		return Entry{}, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.entries(workspace, false)[key]
	return e, ok
}

// Set records the entry for key in workspace. A nil File ignores it.
func (f *File) Set(workspace, key string, e Entry) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	entries := f.entries(workspace, true)
	if old, ok := entries[key]; !ok || old != e {
		entries[key] = e
		f.dirty = true
	}
}

// Delete removes key from workspace, and with prefix set every key that
// starts with key. A nil File ignores it.
func (f *File) Delete(workspace, key string, prefix bool) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	entries := f.entries(workspace, false)
	for k := range entries {
		if k == key || (prefix && strings.HasPrefix(k, key)) {
			delete(entries, k)
			f.dirty = true
		}
	}
	if entries != nil && len(entries) == 0 {
		delete(f.Hosts[f.host], workspace)
		if len(f.Hosts[f.host]) == 0 {
			delete(f.Hosts, f.host)
		}
	}
}

// Save writes the file if anything changed. Keys are sorted, so the file
// diffs and merges cleanly. A nil File does nothing.
func (f *File) Save() error {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.dirty {
		return nil
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return clierr.Wrap(err, clierr.CodeFileNotFound, "")
	}
	if err := os.Rename(tmp, f.path); err != nil {
		os.Remove(tmp)
		return clierr.Wrap(err, clierr.CodeFileNotFound, "")
	}
	f.dirty = false
	return nil
}

// Fingerprint returns a short hash of v's JSON form, leaving out the
// top-level keys in ignore when v is an object. Map keys are sorted by
// encoding/json, so equal values give equal fingerprints whatever their key
// order, and a []string hashes the same as the []any a JSON file decodes to.
func Fingerprint(v any, ignore ...string) string {
	if m, ok := v.(map[string]any); ok && len(ignore) > 0 {
		trimmed := make(map[string]any, len(m))
		maps.Copy(trimmed, m)
		for _, k := range ignore {
			delete(trimmed, k)
		}
		v = trimmed
	}
	data, err := json.Marshal(v)
	if err != nil {
		data = fmt.Append(nil, v)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

// Action is what push does with an item.
type Action string

const (
	Create    Action = "create"    // not in the workspace
	Update    Action = "update"    // push the local copy
	Unchanged Action = "unchanged" // nothing to push
	// Skip: not changed locally, but changed (or deleted) in the workspace
	// since the last pull or push. Pushing the local copy would undo that
	// change, so it is left alone.
	Skip Action = "skip"
	// Conflict: changed both locally and in the workspace since the last
	// pull or push.
	Conflict Action = "conflict"
	// Pending: not changed on either side since a live pull that left the
	// draft's changes out (Entry.Pending). There is nothing to push, and a
	// commit would publish changes that aren't in the local copy.
	Pending Action = "pending"
)

// Decide picks what to do with one item from its current local and remote
// fingerprints (remote is "" when the workspace doesn't have the item) and,
// when hasBase is set, the entry from the last pull or push. Without an
// entry the local copy is pushed whenever it differs: there is nothing to
// tell which side changed. With overwrite, the local copy is pushed even
// over changes made in the workspace.
func Decide(local, remote string, base Entry, hasBase, overwrite bool) Action {
	push := Update
	if remote == "" {
		push = Create
	}
	if overwrite {
		if remote != "" && local == remote {
			return Unchanged
		}
		return push
	}
	if remote == "" {
		if !hasBase {
			return Create
		}
		if local == base.Local {
			if base.Pending && base.Remote == "" {
				return Pending // already missing from the draft at the pull
			}
			return Skip // deleted in the workspace, untouched locally
		}
		return Conflict
	}
	if local == remote {
		return Unchanged
	}
	if !hasBase {
		return Update
	}
	localChanged := local != base.Local
	remoteChanged := remote != base.Remote
	switch {
	case localChanged && remoteChanged:
		return Conflict
	case localChanged && base.Pending:
		return Conflict // the push would replace draft changes the pull left out
	case localChanged:
		return Update
	case remoteChanged:
		return Skip
	case base.Pending:
		return Pending
	}
	// Neither side changed since the last pull or push; they differ only in
	// how the API stores the item.
	return Unchanged
}
