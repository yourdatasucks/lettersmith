package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AnthropicClient struct {
	apiKey string
	model  string
}

type AnthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type AnthropicResponse struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Role    string             `json:"role"`
	Content []AnthropicContent `json:"content"`
	Model   string             `json:"model"`
	Usage   AnthropicUsage     `json:"usage"`
}

type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
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
	promptContent, err := promptTemplates.ReadFile("templates/advocacy-prompt.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt template: %w", err)
	}

	// Convert representatives to RepresentativeOption format
	availableReps := make([]RepresentativeOption, len(req.AvailableRepresentatives))
	for i, rep := range req.AvailableRepresentatives {
		availableReps[i] = RepresentativeOption{
			ID:       rep.ID,
			Name:     rep.Name,
			Title:    rep.Title,
			State:    rep.State,
			Party:    rep.Party,
			District: rep.District,
		}
	}

	data := PromptData{
		Advocacy: AdvocacyContent{
			MainIssue:       req.MainIssue,
			SpecificConcern: req.SpecificIssue,
			RequestedAction: req.RequestedAction,
		},
		Representative: RepresentativeInfo{
			Title: "", // Will be filled after AI selection
			Name:  "",
			State: "",
			Party: "",
		},
		AvailableRepresentatives: availableReps,
		Constituent: ConstituentInfo{
			Name:    req.UserName,
			ZipCode: req.UserZipCode,
		},
		Preferences: LetterPreferences{
			Tone:      req.Tone,
			MaxLength: req.MaxLength,
		},
	}

	tmpl := template.Must(template.New("advocacy").Parse(string(promptContent)))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	prompt := buf.String()

	maxTokens := req.MaxLength * 2
	if maxTokens > 4000 {
		maxTokens = 4000
	}

	anthropicReq := AnthropicRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorBody bytes.Buffer
		errorBody.ReadFrom(resp.Body)

		if resp.StatusCode == 429 {
			return nil, fmt.Errorf("Anthropic rate limit exceeded (429). Error details: %s. Try again in a few minutes", errorBody.String())
		}

		return nil, fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, errorBody.String())
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("no content returned from Anthropic")
	}

	content := anthropicResp.Content[0].Text

	// Parse the selected representative ID and letter content
	selectedRepID, letterContent, selectedRep, err := c.parseAIResponse(content, req.AvailableRepresentatives)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	subject := fmt.Sprintf("Advocacy Letter: %s - %s Constituent", req.MainIssue, selectedRep.State)

	letter := &Letter{
		Subject: subject,
		Content: letterContent,
		Metadata: Metadata{
			Provider:                 "anthropic",
			Model:                    c.model,
			TokensUsed:               anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
			GeneratedAt:              time.Now(),
			Tone:                     req.Tone,
			Theme:                    req.MainIssue,
			MaxLength:                req.MaxLength,
			SelectedRepresentativeID: selectedRepID,
		},
		CreatedAt:              time.Now(),
		SelectedRepresentative: selectedRep,
	}

	return letter, nil
}

func (c *AnthropicClient) parseAIResponse(content string, availableReps []RepresentativeOption) (int, string, *RepresentativeOption, error) {
	lines := strings.Split(content, "\n")

	// Look for the selected representative ID in the first few lines
	selectedRepID := -1
	letterStartIndex := 0

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.Contains(strings.ToUpper(trimmedLine), "SELECTED_REPRESENTATIVE_ID:") {
			// Extract ID from line like "SELECTED_REPRESENTATIVE_ID: 5"
			parts := strings.Split(trimmedLine, ":")
			if len(parts) >= 2 {
				idStr := strings.TrimSpace(parts[1])
				if id, err := strconv.Atoi(idStr); err == nil {
					selectedRepID = id
					letterStartIndex = i + 1
					break
				}
			}
		}
	}

	// Be more strict - don't fall back if we can't parse the ID
	if selectedRepID == -1 {
		return 0, "", nil, fmt.Errorf("could not find SELECTED_REPRESENTATIVE_ID in AI response. Response: %s", content[:min(500, len(content))])
	}

	// Find the selected representative
	var selectedRep *RepresentativeOption
	for _, rep := range availableReps {
		if rep.ID == selectedRepID {
			selectedRep = &rep
			break
		}
	}

	if selectedRep == nil {
		return 0, "", nil, fmt.Errorf("selected representative ID %d not found in available representatives", selectedRepID)
	}

	// Extract letter content (everything after the ID line)
	letterLines := lines[letterStartIndex:]
	letterContent := strings.TrimSpace(strings.Join(letterLines, "\n"))

	if letterContent == "" {
		return 0, "", nil, fmt.Errorf("no letter content found after representative ID")
	}

	// Validate that the letter content mentions the selected representative
	expectedName := selectedRep.Name
	if !strings.Contains(letterContent, expectedName) {
		return 0, "", nil, fmt.Errorf("letter content does not mention selected representative %s (ID: %d). This suggests the AI wrote to a different representative than selected", expectedName, selectedRepID)
	}

	return selectedRepID, letterContent, selectedRep, nil
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
