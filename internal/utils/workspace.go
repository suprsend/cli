package utils

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
)

// CheckWorkspace returns an api-not-found error when the account has no
// workspace with this slug, suggesting the closest one. It returns nil when
// the slug exists or when the workspace list can't be fetched: the command's
// own request then reports whatever is wrong.
func CheckWorkspace(slug string) error {
	if SDKInstance == nil {
		return nil
	}
	err := VerifyWorkspace(slug)
	var ce *clierr.CLIError
	if err != nil && !(errors.As(err, &ce) && ce.Resource == "workspaces/"+slug) {
		log.Debugf("couldn't list workspaces to check %q: %v", slug, err)
		return nil
	}
	return err
}

// VerifyWorkspace is CheckWorkspace that also returns the error from listing
// the workspaces (an invalid token, for example).
func VerifyWorkspace(slug string) error {
	resp, err := SDKInstance.ListWorkspaces(1000, 0)
	if err != nil {
		return clierr.Wrap(err, clierr.CodeAPIInternal, "")
	}
	slugs := make([]string, 0, len(resp.Results))
	for _, ws := range resp.Results {
		if ws.Slug == slug {
			return nil
		}
		slugs = append(slugs, ws.Slug)
	}
	hint := "run `suprsend workspace list` to see the workspaces this token can reach"
	if best := closestSlug(slug, slugs); best != "" {
		hint = fmt.Sprintf("did you mean %q? %s", best, hint)
	}
	return clierr.New(fmt.Sprintf("unknown workspace %q", slug), clierr.CodeAPINotFound).
		WithResource("workspaces/" + slug).
		WithHint(hint)
}

// closestSlug returns the candidate with the smallest edit distance to s, if
// that distance is small enough to be a likely typo.
func closestSlug(s string, candidates []string) string {
	best, bestDist := "", len(s)/2+2
	for _, c := range candidates {
		if d := editDistance(s, c); d < bestDist {
			best, bestDist = c, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
