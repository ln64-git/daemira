/**
 * Notion Utility - Integration with Notion API
 *
 * Features:
 * - Database queries with filtering
 * - Page CRUD operations (create, read, update)
 * - Append content blocks to pages
 * - Sync local files to Notion pages
 * - Retry logic with exponential backoff
 * - Integration with Logger
 */

package utility

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NotionOptions configures the Notion client
type NotionOptions struct {
	LogLevel string // debug, info, warn, error
}

// PageFilter defines filters for database queries
type PageFilter struct {
	Property string
	Value    string
}

// Notion API client with CRUD operations and file syncing
type Notion struct {
	client   *http.Client
	token    string
	logger   *Logger
	baseURL  string
}

// NewNotion creates a new Notion API client
func NewNotion(token string, logger *Logger, options *NotionOptions) (*Notion, error) {
	if token == "" {
		return nil, fmt.Errorf("notion token is required")
	}

	if logger == nil {
		logger = GetLogger()
	}

	n := &Notion{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		token:   token,
		logger: logger,
		baseURL: "https://api.notion.com/v1",
	}

	logger.Info("Notion client initialized")
	return n, nil
}

// QueryDatabaseResponse represents a Notion database query response
type QueryDatabaseResponse struct {
	Results []map[string]interface{} `json:"results"`
	HasMore bool                      `json:"has_more"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

// PageObjectResponse represents a Notion page
type PageObjectResponse map[string]interface{}

// QueryDatabase queries a Notion database with optional filtering
func (n *Notion) QueryDatabase(ctx context.Context, databaseID string, filter *PageFilter) (*QueryDatabaseResponse, error) {
	n.logger.Debug("Querying database: %s", databaseID)

	body := map[string]interface{}{}
	
	if filter != nil && filter.Property != "" && filter.Value != "" {
		body["filter"] = map[string]interface{}{
			"property": filter.Property,
			"rich_text": map[string]interface{}{
				"contains": filter.Value,
			},
		}
	}

	var response QueryDatabaseResponse
	if err := n.makeRequest(ctx, "POST", fmt.Sprintf("/databases/%s/query", databaseID), body, &response); err != nil {
		n.logger.Error("Failed to query database: %v", err)
		return nil, err
	}

	n.logger.Info("Retrieved %d pages from database", len(response.Results))
	return &response, nil
}

// GetPage retrieves a page by ID
func (n *Notion) GetPage(ctx context.Context, pageID string) (*PageObjectResponse, error) {
	n.logger.Debug("Fetching page: %s", pageID)

	var response PageObjectResponse
	if err := n.makeRequest(ctx, "GET", fmt.Sprintf("/pages/%s", pageID), nil, &response); err != nil {
		n.logger.Error("Failed to retrieve page: %v", err)
		return nil, err
	}

	n.logger.Debug("Retrieved page: %s", pageID)
	return &response, nil
}

// CreatePageParams defines parameters for creating a page
type CreatePageParams struct {
	DatabaseID string
	Properties map[string]interface{}
	Content    []map[string]interface{} // Blocks
}

// CreatePage creates a new page in a database
func (n *Notion) CreatePage(ctx context.Context, params CreatePageParams) (*PageObjectResponse, error) {
	n.logger.Debug("Creating page in database: %s", params.DatabaseID)

	body := map[string]interface{}{
		"parent": map[string]interface{}{
			"database_id": params.DatabaseID,
		},
		"properties": params.Properties,
	}

	if len(params.Content) > 0 {
		body["children"] = params.Content
	}

	var response PageObjectResponse
	if err := n.makeRequest(ctx, "POST", "/pages", body, &response); err != nil {
		n.logger.Error("Failed to create page: %v", err)
		return nil, err
	}

	pageID, _ := response["id"].(string)
	n.logger.Info("Created page: %s", pageID)
	return &response, nil
}

// UpdatePage updates an existing page
func (n *Notion) UpdatePage(ctx context.Context, pageID string, properties map[string]interface{}) (*PageObjectResponse, error) {
	n.logger.Debug("Updating page: %s", pageID)

	body := map[string]interface{}{
		"properties": properties,
	}

	var response PageObjectResponse
	if err := n.makeRequest(ctx, "PATCH", fmt.Sprintf("/pages/%s", pageID), body, &response); err != nil {
		n.logger.Error("Failed to update page: %v", err)
		return nil, err
	}

	n.logger.Info("Updated page: %s", pageID)
	return &response, nil
}

// AppendBlocks appends content blocks to a page
func (n *Notion) AppendBlocks(ctx context.Context, pageID string, blocks []map[string]interface{}) error {
	n.logger.Debug("Appending %d blocks to page: %s", len(blocks), pageID)

	body := map[string]interface{}{
		"children": blocks,
	}

	var response map[string]interface{}
	if err := n.makeRequest(ctx, "PATCH", fmt.Sprintf("/blocks/%s/children", pageID), body, &response); err != nil {
		n.logger.Error("Failed to append blocks: %v", err)
		return err
	}

	n.logger.Info("Appended %d blocks to page", len(blocks))
	return nil
}

// SyncFileToPageOptions configures file syncing behavior
type SyncFileToPageOptions struct {
	Overwrite bool
}

// SyncFileToPage syncs local file content to a Notion page as blocks
func (n *Notion) SyncFileToPage(ctx context.Context, pageID, filePath string, options *SyncFileToPageOptions) error {
	n.logger.Info("Syncing file to Notion page: %s -> %s", filePath, pageID)

	content, err := os.ReadFile(filePath)
	if err != nil {
		n.logger.Error("Failed to read file: %v", err)
		return err
	}

	// Convert file content to Notion blocks
	blocks := n.fileContentToBlocks(string(content), filePath)

	if options != nil && options.Overwrite {
		// TODO: Clear existing blocks first
		n.logger.Warn("Overwrite mode not yet implemented")
	}

	if err := n.AppendBlocks(ctx, pageID, blocks); err != nil {
		return err
	}

	n.logger.Info("Successfully synced file to Notion")
	return nil
}

// fileContentToBlocks converts file content to Notion blocks
func (n *Notion) fileContentToBlocks(content, filePath string) []map[string]interface{} {
	blocks := []map[string]interface{}{}

	// Detect file type
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))

	if ext == "md" || ext == "markdown" {
		// Simple markdown parsing
		lines := strings.Split(content, "\n")

		for _, line := range lines {
			line = strings.TrimRight(line, "\r")
			
			if strings.HasPrefix(line, "# ") {
				blocks = append(blocks, map[string]interface{}{
					"object": "block",
					"type":   "heading_1",
					"heading_1": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{
								"text": map[string]interface{}{
									"content": strings.TrimPrefix(line, "# "),
								},
							},
						},
					},
				})
			} else if strings.HasPrefix(line, "## ") {
				blocks = append(blocks, map[string]interface{}{
					"object": "block",
					"type":   "heading_2",
					"heading_2": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{
								"text": map[string]interface{}{
									"content": strings.TrimPrefix(line, "## "),
								},
							},
						},
					},
				})
			} else if strings.HasPrefix(line, "### ") {
				blocks = append(blocks, map[string]interface{}{
					"object": "block",
					"type":   "heading_3",
					"heading_3": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{
								"text": map[string]interface{}{
									"content": strings.TrimPrefix(line, "### "),
								},
							},
						},
					},
				})
			} else if strings.TrimSpace(line) != "" {
				blocks = append(blocks, map[string]interface{}{
					"object": "block",
					"type":   "paragraph",
					"paragraph": map[string]interface{}{
						"rich_text": []map[string]interface{}{
							{
								"text": map[string]interface{}{
									"content": line,
								},
							},
						},
					},
				})
			}
		}
	} else {
		// Plain text - code block
		language := ext
		if language == "" {
			language = "plain text"
		}

		blocks = append(blocks, map[string]interface{}{
			"object": "block",
			"type":   "code",
			"code": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{
						"text": map[string]interface{}{
							"content": content,
						},
					},
				},
				"language": language,
			},
		})
	}

	return blocks
}

// makeRequest performs an HTTP request to the Notion API with retry logic
func (n *Notion) makeRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	return n.retryWrapper(ctx, func() error {
		var reqBody io.Reader
		
		if body != nil {
			jsonData, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %w", err)
			}
			reqBody = bytes.NewBuffer(jsonData)
		}

		url := n.baseURL + endpoint
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+n.token)
		req.Header.Set("Notion-Version", "2022-06-28")
		req.Header.Set("Content-Type", "application/json")

		resp, err := n.client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			var errorResp map[string]interface{}
			if err := json.Unmarshal(respBody, &errorResp); err == nil {
				return fmt.Errorf("notion API error (status %d): %v", resp.StatusCode, errorResp)
			}
			return fmt.Errorf("notion API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		if result != nil {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		return nil
	})
}

// retryWrapper implements exponential backoff retry logic
func (n *Notion) retryWrapper(ctx context.Context, operation func() error) error {
	maxRetries := 3
	baseDelay := 1 * time.Second
	var lastError error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastError = err

		// Don't retry on auth errors or bad requests
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "400") {
			return err
		}

		if attempt < maxRetries {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			n.logger.Warn("Notion API error, retrying in %v (attempt %d/%d): %v", delay, attempt+1, maxRetries, err)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	return lastError
}

