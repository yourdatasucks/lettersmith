package ai

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed templates/*.txt
var promptTemplates embed.FS

type OpenAIClient struct {
	apiKey string
	model  string
}

type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
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

	// Better token calculation: 1 word ≈ 1.33 tokens, with buffer for instructions
	// Add 500 tokens buffer for the representative selection and formatting
	// Be more aggressive with token allocation for longer letters
	baseTokens := int(float64(req.MaxLength) * 1.5)
	bufferTokens := 500

	// For longer letters, add extra buffer to ensure AI doesn't run out of tokens
	if req.MaxLength > 500 {
		bufferTokens = 1000 // Extra buffer for long letters
	}

	maxTokens := baseTokens + bufferTokens

	// Use model-specific limits - be more generous
	var tokenCap int
	switch c.model {
	case "gpt-4", "gpt-4-turbo", "gpt-4-turbo-preview":
		tokenCap = 16000 // Increase significantly for GPT-4 (it can handle up to 128k context)
	case "gpt-3.5-turbo":
		tokenCap = 8000 // Increase for GPT-3.5-turbo
	default:
		tokenCap = 8000 // More generous default
	}

	if maxTokens > tokenCap {
		maxTokens = tokenCap
	}

	// Ensure minimum tokens for any reasonable response
	if maxTokens < 200 {
		maxTokens = 200
	}

	// Debug logging to help troubleshoot word count issues
	log.Printf("OpenAI request: max_length=%d, base_tokens=%d, buffer=%d, final_tokens=%d, model=%s",
		req.MaxLength, baseTokens, bufferTokens, maxTokens, c.model)

	// Create messages with system message for better context setting
	messages := []Message{
		{
			Role:    "system",
			Content: fmt.Sprintf("You are an expert advocacy letter writer. When asked to write a %d-word letter, you MUST write exactly that length. Longer letters require comprehensive, detailed content with multiple well-developed sections. Do not write short letters when long ones are requested.", req.MaxLength),
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	openaiReq := OpenAIRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: 0.7,
	}

	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

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
			return nil, fmt.Errorf("OpenAI rate limit exceeded (429). Error details: %s. Try again in a few minutes or check your quota at https://platform.openai.com/usage", errorBody.String())
		}

		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, errorBody.String())
	}

	var openaiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	content := openaiResp.Choices[0].Message.Content

	// Parse the selected representative ID and letter content
	selectedRepID, letterContent, selectedRep, err := c.parseAIResponse(content, req.AvailableRepresentatives)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	subject := fmt.Sprintf("Advocacy Letter: %s - %s Constituent", req.MainIssue, selectedRep.State)

	// Calculate actual word count for debugging
	actualWordCount := len(strings.Fields(letterContent))

	letter := &Letter{
		Subject: subject,
		Content: letterContent,
		Metadata: Metadata{
			Provider:                 "openai",
			Model:                    c.model,
			TokensUsed:               openaiResp.Usage.TotalTokens,
			GeneratedAt:              time.Now(),
			Tone:                     req.Tone,
			Theme:                    req.MainIssue,
			MaxLength:                req.MaxLength,
			ActualWordCount:          actualWordCount,
			SelectedRepresentativeID: selectedRepID,
		},
		CreatedAt:              time.Now(),
		SelectedRepresentative: selectedRep,
	}

	return letter, nil
}

func (c *OpenAIClient) parseAIResponse(content string, availableReps []RepresentativeOption) (int, string, *RepresentativeOption, error) {
	lines := strings.Split(content, "\n")

	// Look for the selected representative ID in the first few lines
	selectedRepID := -1
	letterStartIndex := 0

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		upperLine := strings.ToUpper(trimmedLine)

		// Look for either "SELECTED_REPRESENTATIVE_ID:" or "SELECTED REPRESENTATIVE ID:"
		if strings.Contains(upperLine, "SELECTED") && strings.Contains(upperLine, "REPRESENTATIVE") && strings.Contains(upperLine, "ID:") {
			// Extract ID from line like "SELECTED_REPRESENTATIVE_ID: 5" or "SELECTED REPRESENTATIVE ID: 5"
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
		// Add debug logging to help troubleshoot parsing issues
		log.Printf("Failed to parse representative ID from AI response. First 200 chars: %s", content[:min(200, len(content))])
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

func (c *OpenAIClient) ValidateAPIKey(ctx context.Context) error {
	if len(c.apiKey) < 20 || !strings.HasPrefix(c.apiKey, "sk-") {
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
