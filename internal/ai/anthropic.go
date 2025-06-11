package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type AnthropicClient struct {
	apiKey string
	model  string
}

func NewAnthropicClient(apiKey, model string) (*AnthropicClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}

	return &AnthropicClient{
		apiKey: apiKey,
		model:  model,
	}, nil
}

func (c *AnthropicClient) GenerateLetter(ctx context.Context, req *GenerationRequest) (*Letter, error) {
	letter := &Letter{
		Subject: fmt.Sprintf("Digital Privacy Advocacy - %s Constituent", req.RepresentativeState),
		Content: c.generatePlaceholderLetter(req),
		Metadata: Metadata{
			Provider:    "anthropic",
			Model:       c.model,
			TokensUsed:  450,
			GeneratedAt: time.Now(),
			Tone:        req.Tone,
			Theme:       "data privacy protection",
			MaxLength:   req.MaxLength,
		},
		CreatedAt: time.Now(),
	}

	return letter, nil
}

func (c *AnthropicClient) ValidateAPIKey(ctx context.Context) error {
	if len(c.apiKey) < 20 || !strings.HasPrefix(c.apiKey, "sk-ant-") {
		return fmt.Errorf("invalid Anthropic API key format")
	}
	return nil
}

func (c *AnthropicClient) GetProviderName() string {
	return "anthropic"
}

func (c *AnthropicClient) EstimateCost(req *GenerationRequest) float64 {
	switch c.model {
	case "claude-3-opus-20240229":
		return 0.08
	case "claude-3-sonnet-20240229":
		return 0.04
	case "claude-3-haiku-20240307":
		return 0.02
	default:
		return 0.04
	}
}

func (c *AnthropicClient) generatePlaceholderLetter(req *GenerationRequest) string {
	return fmt.Sprintf(`Dear %s %s,

As your constituent from %s, I am writing to express my deep concern about the urgent need for comprehensive data privacy legislation in our increasingly digital society.

Every day, millions of Americans unknowingly surrender their most personal information to corporations that profit from this data while providing little transparency or control to consumers. This imbalance of power has created a digital economy where privacy has become a luxury rather than a fundamental right.

I urge you to champion legislation that would:

• Establish meaningful consent requirements that are clear and understandable
• Grant consumers the right to access, correct, and delete their personal data
• Implement data minimization principles limiting collection to necessary purposes
• Create strong accountability measures with real consequences for violations
• Ensure algorithmic transparency in automated decision-making systems

The people of %s are counting on your leadership to protect their digital rights. We need federal privacy standards that put consumers first, not corporate profits.

I would welcome the opportunity to discuss this issue further and learn about your plans to address these critical privacy concerns.

Thank you for your dedicated service and for considering this vital matter.

Respectfully yours,

%s
%s

---
Generated using Lettersmith - Privacy-First Advocacy Platform
Created: %s | Provider: %s (%s)`,
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
