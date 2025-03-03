package gemi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/renniemaharaj/thewriterco-auth-go/pkg/transformer"
)

type Session struct {
	Model *genai.GenerativeModel
}

func (s *Session) SendMessage(ctx context.Context, input *string) (*string, error) {
	session := s.Model.StartChat()

	resp, err := session.SendMessage(ctx, genai.Text(*input))
	if err != nil {
		return nil, fmt.Errorf("error sending message: %v", err)
	}

	response := transformer.PartsToString(resp.Candidates[0].Content.Parts)

	return &response, nil
}

func (s *Session) Ask(ctx context.Context, request *transformer.AskSchema) (*string, error) {
	empty := ""

	if len(request.Conversation) == 0 {
		return &empty, fmt.Errorf("conversation history is empty")
	}

	session := s.Model.StartChat()

	// Set history from previous exchanges (if more than one exists)
	if len(request.Conversation) > 1 {
		session.History = transformer.ExchangesToContent(request.Conversation[:len(request.Conversation)-1])
	}

	jsonBytes, err := json.Marshal(&transformer.AskSchema{
		Conversation:      []transformer.Exchange{request.Conversation[len(request.Conversation)-1]},
		AdditionalContext: request.AdditionalContext,
	})
	if err != nil {
		return &empty, fmt.Errorf("error marshalling request: %v", err)
	}

	// Send the document to the model
	resp, err := session.SendMessage(ctx, genai.Text(string(jsonBytes)))
	if err != nil {
		return &empty, fmt.Errorf("error sending message: %v", err)
	}

	partsStr := transformer.PartsToString(resp.Candidates[0].Content.Parts)
	linted := transformer.LintCodeFences(&partsStr, "json")

	return linted, nil
}
