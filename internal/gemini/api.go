package gemini

import (
	// "bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	// "io"
	"log"
	"strings"

	routing "github.com/go-ozzo/ozzo-routing/v2"
	// "github.com/vmihailenco/msgpack"
	"github.com/vmihailenco/msgpack/v5"
)

// RegisterHandlers registers the routes for the GenAI API.
func RegisterHandlers(r *routing.RouteGroup, service *Service) {
	r.Post("/ask", handleAsk(service))
	r.Post("/find", handleFind(service))

}

// Struct representing the Scripture schema
type Scripture struct {
	Book         string `json:"book"`
	ChapterNo    int    `json:"chapterNo"`
	VerseNo      int    `json:"verseNo"`
	VerseContent string `json:"verseContent"`
}

// Struct representing the overall response schema
type ResponseSchema struct {
	MarkupResponse string      `json:"markupResponse"` // For the HTML markup as a string
	DataObjects    []Scripture `json:"dataObjects"`    // Array of Scripture structs
}

// RemoveCodeFences removes ```json from the start and ``` from the end of the input string.
func RemoveCodeFences(input string) string {
	const codeFenceStart = "```json"
	const codeFenceEnd = "```"

	// Trim the starting "```json"
	input = strings.TrimPrefix(input, codeFenceStart)

	// Trim any leading/trailing whitespace or newlines to better detect the ending code fence
	input = strings.TrimSpace(input)

	// Trim the ending "```"
	input = strings.TrimSuffix(input, codeFenceEnd)

	// Trim excess whitespace again
	return strings.TrimSpace(input)
}

// Function to parse JSON into the ResponseSchema struct
func ParseResponse(jsonData string) (*ResponseSchema, error) {
	// Create an empty ResponseSchema instance
	var response ResponseSchema

	// Unmarshal the input JSON into the struct
	err := json.Unmarshal([]byte(RemoveCodeFences(jsonData)), &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

type Exchange struct {
	Sender  string `json:"sender"`
	Content string `json:"content"`
}

type AskSchema struct {
	Conversation      []Exchange `json:"conversation"`
	AdditionalContext []string   `json:"additionalContext"`
}

// handleAsk handles requests for the /ask endpoint.
func handleAsk(service *Service) routing.Handler {
	return func(c *routing.Context) error {
		var rawRequest struct {
			Message string `json:"message"` // Full JSON object as a string
		}

		// Read request
		if err := c.Read(&rawRequest); err != nil {
			log.Printf("Failed to read request: %v", err)
			return c.Write(map[string]string{"error": "Invalid request format"})
		}

		// Parse the JSON string inside `rawRequest.Message`
		var askSchema struct {
			Conversation      string   `json:"conversation"` // Base64-encoded MessagePack
			AdditionalContext []string `json:"additionalContext"`
		}

		if err := json.Unmarshal([]byte(rawRequest.Message), &askSchema); err != nil {
			log.Printf("Failed to parse JSON string: %v", err)
			return c.Write(map[string]string{"error": "Invalid JSON format"})
		}

		// ✅ Log received Base64 data
		// log.Printf("Base64 Conversation: %s", askSchema.Conversation)

		// Decode Base64 string into MessagePack bytes
		msgPackData, err := base64.StdEncoding.DecodeString(askSchema.Conversation)
		if err != nil {
			log.Printf("Failed to decode Base64: %v", err)
			return c.Write(map[string]string{"error": "Invalid Base64 encoding"})
		}

		// ✅ Log decoded MessagePack bytes
		// log.Printf("Decoded MessagePack Bytes: %v", msgPackData)

		// First, unmarshal into an `interface{}` to inspect structure
		var intermediate interface{}
		if err := msgpack.Unmarshal(msgPackData, &intermediate); err != nil {
			log.Printf("Failed to parse MessagePack (raw output): %v", err)
			return c.Write(map[string]string{"error": "Invalid MessagePack format"})
		}

		// ✅ Log the raw decoded structure
		// log.Printf("Raw decoded MessagePack output: %+v", intermediate)

		// Ensure correct type conversion
		var conversation []Exchange
		if arr, ok := intermediate.([]interface{}); ok {
			conversation = make([]Exchange, len(arr))
			for i, v := range arr {
				if obj, ok := v.(map[string]interface{}); ok {
					conversation[i] = Exchange{
						Sender:  obj["sender"].(string),
						Content: obj["content"].(string),
					}
				}
			}
		} else {
			log.Printf("Unexpected MessagePack structure: %T", intermediate)
			return c.Write(map[string]string{"error": "Unexpected data format"})
		}

		// ✅ Log successfully parsed conversation
		// log.Printf("Parsed Conversation: %+v", conversation)

		// Build final request object
		// Convert request to JSON string
		jsonBytes, err := json.Marshal(AskSchema{
			Conversation:      conversation,
			AdditionalContext: askSchema.AdditionalContext,
		})
		if err != nil {
			return c.Write(map[string]string{"error": "Failed to marshal request"})
		}

		// Send the structured JSON request
		resp, err := service.SendMessage(context.Background(), string(jsonBytes))

		if err != nil {
			return c.Write(map[string]string{"error": err.Error()})
		}

		// Send response back to frontend
		return c.Write(map[string]interface{}{
			"response": RemoveCodeFences(strings.Join(resp, "\n")),
		})
	}
}

// handleFind handles requests for the /find endpoint.
func handleFind(service *Service) routing.Handler {

	return func(c *routing.Context) error {
		var request struct {
			Message string `json:"message"`
		}
		if err := c.Read(&request); err != nil {
			return c.Write(map[string]string{"error": "Invalid request"})
		}

		// Define a custom prompt to locate Bible verses.
		prompt := fmt.Sprintf(`
	{
  		"specializedTask": "-@escapeMarkup Locate contextually matching and relevant Bible (KJV) scriptures",
  		"responseSchema": [
    		{
      			"book": "VALID_BIBLE_BOOK_NAME",
      			"chapterNo": "INTEGER",
      			"verseNo": "INTEGER",
      			"verseContent": "STRING"
    		}
  		],
  		"userProvidedContext": "%s"
	}
	`, request.Message)

		if request.Message == "" {
			return c.Write(map[string]string{"error": "Query is required."})
		}

		// Call the service to find relevant Bible verses.
		resp, err := service.SendMessage(context.Background(), prompt)
		if err != nil {
			return c.Write(map[string]string{"error": err.Error()})
		}

		dataObjects, err := ParseResponse(strings.Join(resp, "\n"))
		if err != nil {
			return c.Write(map[string]string{"error": err.Error()})
		}

		// Convert dataObjects to JSON string
		jsonStr, err := json.Marshal(dataObjects)
		if err != nil {
			return c.Write(map[string]string{"error": "Failed to serialize response"})
		}
		return c.Write(map[string]interface{}{"response": string(jsonStr)})
	}
}
