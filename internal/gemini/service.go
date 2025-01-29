package gemini

import (
	"context"
	"fmt"
	"log"

	// "os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	// "github.com/qiangxue/go-rest-api/internal/gemini"
	// "github.com/qiangxue/go-rest-api/pkg/log"
	"google.golang.org/api/option"
)

// Config holds configuration for the GenAI client.
type Config struct {
	APIKey           string
	ModelName        string
	Temperature      float32
	TopK             int32
	TopP             float32
	MaxOutputTokens  int32
	ResponseMIMEType string
}

// Service wraps the GenAI client and provides methods to interact with it.
type Service struct {
	client *genai.Client
	model  *genai.GenerativeModel
	config Config
}

// NewService creates a new instance of Service with the provided config.
func NewService(ctx context.Context, config Config) *Service {

	if config.APIKey == "" || config.ModelName == "" {
		log.Fatalf("APIKey and ModelName are required")
	}

	if config.APIKey == "" || config.ModelName == "" {
		log.Fatalf("APIKey and ModelName are required")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(config.APIKey))
	if err != nil {
		log.Fatalf("failed to create GenAI client: %v", err)
	}

	model := client.GenerativeModel(config.ModelName)
	if model == nil {
		log.Fatalf("invalid model name: %s", config.ModelName)
	}

	// Configure model
	model.SetTemperature(config.Temperature)
	model.SetTopK(config.TopK)
	model.SetTopP(config.TopP)
	model.SetMaxOutputTokens(config.MaxOutputTokens)
	model.ResponseMIMEType = config.ResponseMIMEType
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(GetAxiomsAndInstructions())},
	}

	return &Service{
		client: client,
		model:  model,
		config: config,
		// logger: logger,
	}
}

// SendMessage sends a message to the generative model and returns the response.
func (s *Service) SendMessage(ctx context.Context, input string) ([]string, error) {
	if s.model == nil {
		return nil, fmt.Errorf("model is not initialized")
	}

	fmt.Println("Recieved input: ", input)

	session := s.model.StartChat()

	fmt.Println("Sending request to genai: ", input)

	resp, err := session.SendMessage(ctx, genai.Text(input))
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned in response")
	}

	// Collect response parts
	var messages []string
	for _, part := range resp.Candidates[0].Content.Parts {
		switch v := part.(type) {
		case genai.Text:
			messages = append(messages, string(v))
		default:
			// s.logger.Errorf("unexpected part type: %T", v)
		}
	}

	fmt.Println("Recieved response from model: ", strings.Join(messages, "\n"))

	return messages, nil
}

// Close closes the GenAI client.
func (s *Service) Close() error {
	if s.client == nil {
		return fmt.Errorf("client is not initialized")
	}
	return s.client.Close()
}
