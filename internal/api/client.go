package api

import (
	"context"
	"fmt"
	"time"
)

type Letter struct {
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	Metadata  Metadata  `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
}

type Metadata struct {
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	TokensUsed  int       `json:"tokens_used"`
	GeneratedAt time.Time `json:"generated_at"`
	Tone        string    `json:"tone"`
	Theme       string    `json:"theme"`
	MaxLength   int       `json:"max_length"`
}

type GenerationRequest struct {
	UserName            string   `json:"user_name"`
	UserZipCode         string   `json:"user_zip_code"`
	RepresentativeName  string   `json:"representative_name"`
	RepresentativeTitle string   `json:"representative_title"`
	RepresentativeState string   `json:"representative_state"`
	Themes              []string `json:"themes"`
	Tone                string   `json:"tone"`
	MaxLength           int      `json:"max_length"`
	Context             string   `json:"context,omitempty"`
}

type AIClient interface {
	GenerateLetter(ctx context.Context, req *GenerationRequest) (*Letter, error)
	ValidateAPIKey(ctx context.Context) error
	GetProviderName() string
	EstimateCost(req *GenerationRequest) float64
}

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
