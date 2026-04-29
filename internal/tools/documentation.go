package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func searchDocsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	encodedQuery := url.QueryEscape(query)
	response, err := http.Get(fmt.Sprintf("https://rag.suprsend.com/?query=%s", encodedQuery))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(body)), nil
}

func fetchDocsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uri, err := request.RequireString("uri")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !strings.HasSuffix(uri, ".md") {
		uri = uri + ".md"
	}
	url := fmt.Sprintf("https://docs.suprsend.com/%s", uri)
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(body)), nil
}

func newDocumentationTools() []*Tool {
	searchDoc := &Tool{
		Name:        "documentation.search",
		Description: "Enables querying SuprSend documentation",
		MCPTool: mcp.NewTool("search_suprsend_documentation",
			mcp.WithDescription(`Search SuprSend's product documentation for technical guidance — APIs, SDKs, workflows, templates, tenants, lists, vendors, and connectors.

When to use:
- The user asks how a SuprSend feature works or how to integrate one.
- You need to verify a behavior before calling a write tool.
- You're debugging an integration error.

When NOT to use: for runtime operations on SuprSend resources (users, objects, tenants, workflows) — those have dedicated tools.

Returns: a JSON array of {uri, snippet}. Snippets are excerpts; if a snippet doesn't fully answer, follow up with fetch_suprsend_documentation on the relevant uri.

Tips: use precise technical terms ("workflow trigger conditions", not "the rule thing"); add synonyms if the first query returns nothing.`),
			mcp.WithString("query",
				mcp.Description(`Search query. The query should: 
					- Identify the core concepts and intent 
					- Add relevant synonyms and related terms 
					- Structure the query to emphasize key terms 
					- Include technical or domain-specific terminology if applicable`),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		Handler: searchDocsHandler,
	}

	fetchDoc := &Tool{
		Name:        "documentation.fetch",
		Description: "Fetch the full documentation content for the given uri.",
		MCPTool: mcp.NewTool("fetch_suprsend_documentation",
			mcp.WithDescription(`Fetch the full content of a SuprSend documentation page when a snippet from search_suprsend_documentation is insufficient.

When to use: after search_suprsend_documentation, when the snippet excerpt doesn't fully answer and you need surrounding context, code examples, or full reference material.

When NOT to use: to discover documentation — search first; don't construct uris yourself.

Returns: the page contents as markdown.`),
			mcp.WithString("uri",
				mcp.Description(`The uri of the documentation to fetch.`),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		Handler: fetchDocsHandler,
	}

	return []*Tool{searchDoc, fetchDoc}
}

func init() {
	for _, t := range newDocumentationTools() {
		RegisterTool(t, "documentation")
	}
}
