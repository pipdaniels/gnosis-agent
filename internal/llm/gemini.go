package llm

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// GeminiClient wraps Google Gemini API for LLM-powered text generation
type GeminiClient struct {
	client *genai.Client
	model  string
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(ctx context.Context, apiKey, model string) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	
	if model == "" {
		model = "gemini-1.5-pro" // Default model
	}
	
	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

// GenerateText generates text from a prompt using Gemini
func (c *GeminiClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	contents := []*genai.Content{
		genai.NewContentFromText(prompt, genai.RoleUser),
	}
	
	resp, err := c.client.Models.GenerateContent(ctx, c.model, contents, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}
	
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response candidates from Gemini")
	}
	
	if len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content parts in Gemini response")
	}
	
	// Extract text from the first part
	return fmt.Sprint(resp.Candidates[0].Content.Parts[0]), nil
}

// GenerateWithSystemPrompt generates text with a system instruction
func (c *GeminiClient) GenerateWithSystemPrompt(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	contents := []*genai.Content{
		genai.NewContentFromText(userPrompt, genai.RoleUser),
	}
	
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
	}
	
	resp, err := c.client.Models.GenerateContent(ctx, c.model, contents, config)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}
	
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}
	
	return fmt.Sprint(resp.Candidates[0].Content.Parts[0]), nil
}

