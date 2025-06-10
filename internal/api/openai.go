package api

import (
	"context"
	"fmt"
	"time"
)

type OpenAIClient struct {
	apiKey string
	model  string
}

func NewOpenAIClient(apiKey, model string) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	if model == "" {
		model = "gpt-4"
	}

	return &OpenAIClient{
		apiKey: apiKey,
		model:  model,
	}, nil
}

func (c *OpenAIClient) GenerateLetter(ctx context.Context, req *GenerationRequest) (*Letter, error) {
	letter := &Letter{
		Subject: fmt.Sprintf("Privacy Rights Advocacy - %s Resident", req.RepresentativeState),
		Content: c.generatePlaceholderLetter(req),
		Metadata: Metadata{
			Provider:    "openai",
			Model:       c.model,
			TokensUsed:  500,
			GeneratedAt: time.Now(),
			Tone:        req.Tone,
			Theme:       "data privacy protection",
			MaxLength:   req.MaxLength,
		},
		CreatedAt: time.Now(),
	}

	return letter, nil
}

func (c *OpenAIClient) ValidateAPIKey(ctx context.Context) error {
	if len(c.apiKey) < 20 || !startsWith(c.apiKey, "sk-") {
		return fmt.Errorf("invalid OpenAI API key format")
	}
	return nil
}

func (c *OpenAIClient) GetProviderName() string {
	return "openai"
}

func (c *OpenAIClient) EstimateCost(req *GenerationRequest) float64 {
	switch c.model {
	case "gpt-4":
		return 0.05
	case "gpt-3.5-turbo":
		return 0.01
	default:
		return 0.03
	}
}

func (c *OpenAIClient) generatePlaceholderLetter(req *GenerationRequest) string {
	return fmt.Sprintf(`Dear %s %s,

I am writing to you as a concerned constituent from %s to urge your support for stronger data privacy protections and consumer rights in our digital age.

As technology companies continue to collect vast amounts of personal data from American citizens, it has become increasingly clear that our current privacy laws are inadequate to protect consumers from data misuse, unauthorized sharing, and corporate surveillance.

I believe it is crucial that Congress takes immediate action to:

1. Establish comprehensive federal privacy legislation that gives consumers meaningful control over their personal data
2. Require clear, understandable consent processes for data collection and sharing
3. Implement strong enforcement mechanisms with significant penalties for violations
4. Ensure transparency in how companies use and monetize personal information

The residents of %s deserve to know that their elected representatives are working to protect their fundamental right to privacy in the digital realm. I respectfully urge you to support legislation that puts consumer privacy rights first.

Thank you for your time and consideration. I look forward to your response and your leadership on this critical issue.

Sincerely,
%s
Constituent, %s

---
This letter was generated using Lettersmith, a privacy-focused advocacy tool.
Generated on %s using %s (%s)`,
		req.RepresentativeTitle,
		req.RepresentativeName,
		req.UserZipCode,
		req.RepresentativeState,
		req.UserName,
		req.UserZipCode,
		time.Now().Format("January 2, 2006"),
		c.GetProviderName(),
		c.model,
	)
}
