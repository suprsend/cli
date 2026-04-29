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
			mcp.WithDescription(`Use this tool to get technical guidance or answers related to SuprSend. It is especially helpful when:
				- You have questions about SuprSend's capabilities, features, or integrations (e.g., Workflows, Templates, Tenants, Lists, Vendors, Connectors, etc.).
				- You are unsure how a specific SuprSend functionality or API works.
				- You are writing or debugging an integration with SuprSend.
			How to use:
				- Frame your queries using precise technical terms; avoid vague language.
				- The tool returns a JSON array, where each item contains:
				- uri: The documentation path.
				- snippet: A relevant excerpt from the documentation.
			Answering process:
				- Review all provided snippets to answer the question.
				- If the snippets are insufficient, use the corresponding uri to fetch the full documentation with the fetch_suprsend_documentation tool.
				- Process each resource in the order provided.
				- Use information from both the snippets and the full documentation to construct your final answer.`),
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
			mcp.WithDescription(`Use this tool to fetch the full documentation content for the given uri.`),
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
