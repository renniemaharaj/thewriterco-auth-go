package gemini

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

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

func RawToAskSchema(raw *transformer.RawAskSchema) (*transformer.AskSchema, error) {

	msgPackData, err := base64.StdEncoding.DecodeString(raw.Conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Base64: %w", err)
	}

	log.Printf("Decoded Base64: %s", msgPackData)

	var intermediate interface{}
	if err := msgpack.Unmarshal(msgPackData, &intermediate); err != nil {
		return nil, fmt.Errorf("failed to parse MessagePack (raw output): %w", err)
	}

	// Ensure correct type conversion
	var conversation []transformer.Exchange
	arr, ok := intermediate.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected MessagePack structure: %T", intermediate)
	}

	conversation = make([]transformer.Exchange, len(arr))
	for i, v := range arr {
		if obj, ok := v.(map[string]interface{}); ok {
			conversation[i] = transformer.Exchange{
				Role:    obj["role"].(string),
				Content: obj["content"].(string),
			}
		}
	}

	return &transformer.AskSchema{
		Conversation:      conversation,
		AdditionalContext: raw.AdditionalContext,
	}, nil
}
