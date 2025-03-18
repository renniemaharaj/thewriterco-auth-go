package gemini

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/generative-ai-go/genai"
	"github.com/renniemaharaj/thewriterco-auth-go/internal/gemini/blocks"
	"github.com/renniemaharaj/thewriterco-auth-go/pkg/transformer"
	"github.com/vmihailenco/msgpack/v5"
)

// function to parse JSON into the ResponseSchema struct
func ParseResponse(jsonData string) (interface{}, error) {
	// create an empty ResponseSchema instance
	var response interface{}

	// unmarshal the input JSON into the struct
	err := json.Unmarshal([]byte(*transformer.LintCodeFences(&jsonData, "json")), &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// func RequestMessageToAskSchema(message string) (*ChatSchema, error) {
// 	var askSchema RawAskSchema

// 	err := json.Unmarshal([]byte(message), &askSchema)
// 	if err != nil {
// 		log.Printf("Failed to parse JSON string: %v", err)
// 		return nil, fmt.Errorf("failed to parse JSON string: %w", err)
// 		// return c.Write(map[string]string{"response": "Invalid JSON format"})
// 	}

// 	treatedSchema, err := RawToAskSchema(&askSchema)

// 	if err != nil {
// 		log.Printf("Failed to process AskSchema: %v", err)
// 		return nil, fmt.Errorf("failed to process AskSchema: %w", err)
// 		// return c.Write(map[string]string{"response": "Failed to process request"})
// 	}

//		return treatedSchema, nil
//	}
type structMessage struct {
	Base64  string              `json:"encoded"`
	Context []map[string]string `json:"context"`
}

func DecodeConversation(message string) (*ChatSchema, error) {

	// log.Printf("Decoding message: %s\n", message)
	msg := structMessage{}
	var chatSchema ChatSchema

	err := json.Unmarshal([]byte(message), &msg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON string: %w", err)
	}

	// log.Printf("Decoded message!: %v\n", msg)

	// decoded := []byte(msg.Encoded)
	decoded, err := base64.StdEncoding.DecodeString(msg.Base64)
	if err != nil {
		fmt.Println("Base64 Decoding Error:", err)
		return nil, fmt.Errorf("failed to decode Base64: %w", err)
	}

	// log.Printf("Encoded Base64: %s\n", decoded)

	var conversation []map[string]string
	err = msgpack.Unmarshal(decoded, &conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MessagePack (raw output): %w", err)
	}

	// log.Printf("Decoded conversation: %v\n", conversation)

	chatSchema.Context = msg.Context
	var exchanges []Exchange
	for _, conv := range conversation {
		exchanges = append(exchanges, Exchange{
			Role:    conv["role"],
			Content: conv["content"],
		})
	}
	chatSchema.Conversation = exchanges

	// log.Printf("Decoded message (finish) %v\n", chatSchema)
	return &chatSchema, nil
}

func ExchangesToHistory(conversation []Exchange) []*genai.Content {
	history := []*genai.Content{}

	for idx, part := range conversation {
		if idx == len(conversation)-1 {
			break
		}
		history = append(history, &genai.Content{
			Role: part.Role,

			Parts: []genai.Part{
				genai.Text(part.Content),
			},
		})
	}
	return history
}

func StrayHTMLToResponseSchema(resp string) *blocks.ResponseSchema {
	warning := "<p style='display:none;' recipient='model'>-@here Warning. You broke the response schema with this stray html response, but was recovered</p>"
	content := resp + "<br><br>" + warning
	return &blocks.ResponseSchema{
		ResponseBlocks: []blocks.ResponseBlock{
			{
				BlockPrimitive: blocks.BlockPrimitive{
					Type: "markup",
				},
				Content: blocks.Markup{
					BlockPrimitive: blocks.BlockPrimitive{
						Type: "markup",
					},
					MarkupContent: content,
				},
			},
		},
	}
}

func isValidHTMLStructure(resp string) bool {
	trimmed := strings.TrimSpace(resp)
	return len(resp) > 0 && ((strings.HasPrefix(trimmed, "<div") && strings.HasSuffix(trimmed, "</div>")) ||
		(strings.HasPrefix(trimmed, "<p") && strings.HasSuffix(trimmed, "</p>")))
}

func ValidateResponseSchema(resp string) error {
	log.Println("Validating response...")

	var r blocks.ResponseSchema

	err := json.Unmarshal([]byte(resp), &r)
	if err != nil {
		// Fallback check for basic HTML structure
		if isValidHTMLStructure(resp) {
			log.Println("JSON validation failed but found valid HTML structure")
			return nil
		}
		log.Printf("JSON Unmarshal error: %v\n", err)
		return err
	}

	err = validation.ValidateStruct(&r,
		validation.Field(&r.ResponseBlocks, validation.Required, validation.Each(validation.NotNil)),
	)

	if err != nil {
		log.Printf("Validation failed: %v\n", err)
		return err
	}

	log.Println("Validation passed!")
	return nil
}
