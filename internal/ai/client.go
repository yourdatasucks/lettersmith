package ai

import (
	"context"
	"fmt"
	"time"
)

type Letter struct {
	Subject                string                `json:"subject"`
	Content                string                `json:"content"`
	Metadata               Metadata              `json:"metadata"`
	CreatedAt              time.Time             `json:"created_at"`
	SelectedRepresentative *RepresentativeOption `json:"selected_representative"`
}

type Metadata struct {
	Provider                 string    `json:"provider"`
	Model                    string    `json:"model"`
	TokensUsed               int       `json:"tokens_used"`
	GeneratedAt              time.Time `json:"generated_at"`
	Tone                     string    `json:"tone"`
	Theme                    string    `json:"theme"`
	MaxLength                int       `json:"max_length"`
	SelectedRepresentativeID int       `json:"selected_representative_id"`
}

type GenerationRequest struct {
	MainIssue                string                 `json:"main_issue"`
	SpecificIssue            string                 `json:"specific_issue"`
	RequestedAction          string                 `json:"requested_action"`
	UserName                 string                 `json:"user_name"`
	UserZipCode              string                 `json:"user_zip_code"`
	AvailableRepresentatives []RepresentativeOption `json:"available_representatives"`
	Tone                     string                 `json:"tone"`
	MaxLength                int                    `json:"max_length"`
}

type PromptData struct {
	Advocacy                 AdvocacyContent        `json:"advocacy"`
	Representative           RepresentativeInfo     `json:"representative"`
	AvailableRepresentatives []RepresentativeOption `json:"available_representatives"`
	Constituent              ConstituentInfo        `json:"constituent"`
	Preferences              LetterPreferences      `json:"preferences"`
}

type RepresentativeOption struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Title    string  `json:"title"`
	State    string  `json:"state"`
	Party    *string `json:"party,omitempty"`
	District *string `json:"district,omitempty"`
}

type AdvocacyContent struct {
	MainIssue       string `json:"main_issue"`
	SpecificConcern string `json:"specific_concern"`
	RequestedAction string `json:"requested_action"`
}

type RepresentativeInfo struct {
	Title string `json:"title"`
	Name  string `json:"name"`
	State string `json:"state"`
	Party string `json:"party"`
}

type ConstituentInfo struct {
	Name    string `json:"name"`
	ZipCode string `json:"zip_code"`
}

type LetterPreferences struct {
	Tone      string `json:"tone"`
	MaxLength int    `json:"max_length"`
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

// Helper function used by both AI clients
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
