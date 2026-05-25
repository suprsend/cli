package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/suprsend/cli/pkg/mcpsdk"
)

// NOTE: handlers in this file hit external endpoints (rag.suprsend.com and
// docs.suprsend.com) rather than the SuprSend management API, so the
// IsAuthError → MarkSessionDead discipline applied to ported SuprSend-API
// handlers does not apply here — there is no service-token auth path to
// invalidate. If a handler is ever rewritten to go through
// utils.GetSuprSendWorkspaceClient, add the standard auth-check guard.

func searchDocsHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	query, err := args.RequireString("query")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	encodedQuery := url.QueryEscape(query)
	response, err := http.Get(fmt.Sprintf("https://rag.suprsend.com/?query=%s", encodedQuery))
	if err != nil {
		return mcpsdk.Result{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return mcpsdk.Result{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return mcpsdk.Result{}, err
	}

	return mcpsdk.Result{Text: string(body)}, nil
}

func fetchDocsHandler(ctx context.Context, args mcpsdk.Args) (mcpsdk.Result, error) {
	uri, err := args.RequireString("uri")
	if err != nil {
		return mcpsdk.Result{Text: err.Error(), IsError: true}, nil
	}

	if !strings.HasSuffix(uri, ".md") {
		uri = uri + ".md"
	}
	docURL := fmt.Sprintf("https://docs.suprsend.com/%s", uri)
	response, err := http.Get(docURL)
	if err != nil {
		return mcpsdk.Result{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return mcpsdk.Result{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return mcpsdk.Result{}, err
	}

	return mcpsdk.Result{Text: string(body)}, nil
}

func newDocumentationTools() []*Tool {
	searchDoc := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "search_suprsend_documentation",
			Description: `Search SuprSend's product documentation for technical guidance — APIs, SDKs, workflows, templates, tenants, lists, vendors, and connectors.

When to use:
- The user asks how a SuprSend feature works or how to integrate one.
- You need to verify a behavior before calling a write tool.
- You're debugging an integration error.

When NOT to use: for runtime operations on SuprSend resources (users, objects, tenants, workflows) — those have dedicated tools.

Returns: a JSON array of {uri, snippet}. Snippets are excerpts; if a snippet doesn't fully answer, follow up with fetch_suprsend_documentation on the relevant uri.

Tips: use precise technical terms ("workflow trigger conditions", not "the rule thing"); add synonyms if the first query returns nothing.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"query": {
						Type: "string",
						Description: `Search query. The query should:
- Identify the core concepts and intent
- Add relevant synonyms and related terms
- Structure the query to emphasize key terms
- Include technical or domain-specific terminology if applicable`,
					},
				},
				Required: []string{"query"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  mcpsdk.BoolPtr(true),
			},
			Handler: searchDocsHandler,
		},
	}

	fetchDoc := &Tool{
		Tool: &mcpsdk.Tool{
			Name: "fetch_suprsend_documentation",
			Description: `Fetch the full content of a SuprSend documentation page when a snippet from search_suprsend_documentation is insufficient.

When to use: after search_suprsend_documentation, when the snippet excerpt doesn't fully answer and you need surrounding context, code examples, or full reference material.

When NOT to use: to discover documentation — search first; don't construct uris yourself.

Returns: the page contents as markdown.`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"uri": {
						Type:        "string",
						Description: `The uri of the documentation to fetch.`,
					},
				},
				Required: []string{"uri"},
			},
			Annotations: mcpsdk.Annotations{
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  mcpsdk.BoolPtr(true),
			},
			Handler: fetchDocsHandler,
		},
	}

	return []*Tool{searchDoc, fetchDoc}
}

func init() {
	for _, t := range newDocumentationTools() {
		RegisterTool(t, "documentation")
	}
}
