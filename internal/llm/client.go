package llm

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Client struct {
	client *anthropic.Client
	model  string
}

type Config struct {
	APIKey string
	Model  string
}

// NewClient creates a new Claude API client
func NewClient() (*Client, error) {
	apiKey := os.Getenv("CLAUDE_TOKEN")
	if apiKey == "" {
		return nil, fmt.Errorf("CLAUDE_TOKEN environment variable is required")
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "claude-sonnet-4-5-20250929" // Default to Claude Sonnet 4.5
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &Client{
		client: &client,
		model:  model,
	}, nil
}

// SendMessage sends a message to Claude and returns the response
func (c *Client) SendMessage(ctx context.Context, systemPrompt string, userMessage string, maxTokens int) (string, error) {
	if maxTokens == 0 {
		maxTokens = 4096 // Default max tokens
	}

	// Create the message request
	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{
			anthropic.TextBlockParam{
				Type: "text",
				Text: systemPrompt,
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(userMessage),
			),
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to send message to Claude: %w", err)
	}

	// Extract text from response
	if len(message.Content) == 0 {
		return "", fmt.Errorf("no content in Claude response")
	}

	// Get the first text block
	textBlock := message.Content[0].Text
	return textBlock, nil
}

// SendMessageWithRetry sends a message with exponential backoff retry
func (c *Client) SendMessageWithRetry(ctx context.Context, systemPrompt string, userMessage string, maxTokens int, maxRetries int) (string, error) {
	if maxRetries == 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		response, err := c.SendMessage(ctx, systemPrompt, userMessage, maxTokens)
		if err == nil {
			return response, nil
		}

		lastErr = err
	}

	return "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}
