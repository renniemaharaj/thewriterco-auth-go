package gemi

import (
	"context"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/renniemaharaj/thewriterco-auth-go/pkg/transformer"
)

func Model(ctx context.Context, apiKey string, base string) (*genai.GenerativeModel, func(), error) {
	log.Println("Creating google gemini client...")

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, nil, fmt.Errorf("error creating client: %v", err)
	}

	// model := client.GenerativeModel("gemini-2.0-pro-exp-02-05")
	model := client.GenerativeModel(base)
	model.SetTemperature(1)
	model.SetTopK(64)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "text/plain"
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text(transformer.GetProgramming()),
		},
	}

	return model, func() { client.Close() }, nil
}
