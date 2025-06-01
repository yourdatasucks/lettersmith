package api

import (
	"context"
	"fmt"
	"time"
)

// Letter represents a generated letter
type Letter struct {
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	Metadata  Metadata  `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
}

// Metadata contains information about letter generation
type Metadata struct {
	Provider    string    `json:"provider"`    // openai, anthropic
	Model       string    `json:"model"`       // gpt-4, claude-3-sonnet, etc.
	TokensUsed  int       `json:"tokens_used"` // approximate token usage
	GeneratedAt time.Time `json:"generated_at"`
	Tone        string    `json:"tone"`       // professional, passionate, etc.
	Theme       string    `json:"theme"`      // privacy, consumer protection, etc.
	MaxLength   int       `json:"max_length"` // target word count
}

// GenerationRequest contains parameters for letter generation
type GenerationRequest struct {
	UserName            string   `json:"user_name"`
	UserZipCode         string   `json:"user_zip_code"`
	RepresentativeName  string   `json:"representative_name"`
	RepresentativeTitle string   `json:"representative_title"`
	RepresentativeState string   `json:"representative_state"`
	Themes              []string `json:"themes"`
	Tone                string   `json:"tone"`
	MaxLength           int      `json:"max_length"`
	Context             string   `json:"context,omitempty"` // Additional context for generation
}

// AIClient interface for different AI providers
type AIClient interface {
	// GenerateLetter creates a personalized letter based on the request
	GenerateLetter(ctx context.Context, req *GenerationRequest) (*Letter, error)

	// ValidateAPIKey checks if the API key is valid
	ValidateAPIKey(ctx context.Context) error

	// GetProviderName returns the name of the AI provider
	GetProviderName() string

	// EstimateCost returns the estimated cost for generating a letter (in USD)
	EstimateCost(req *GenerationRequest) float64
}

// NewClient creates a new AI client based on the provider
func NewClient(provider, apiKey, model string) (AIClient, error) {
	switch provider {
	case "openai":
		return NewOpenAIClient(apiKey, model)
	case "anthropic":
		return NewAnthropicClient(apiKey, model)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", provider)
	}
}
