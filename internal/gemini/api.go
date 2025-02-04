package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	routing "github.com/go-ozzo/ozzo-routing/v2"
	"github.com/vmihailenco/msgpack/v5"
)

// RegisterHandlers registers the routes for the GenAI API.
func RegisterHandlers(r *routing.RouteGroup, service *Service) {
	r.Post("/ask", handleAsk(service))
	r.Post("/find", handleFind(service))
	r.Post("/genealogy", handleGenealogy(service))

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

type AskSchema struct {
	Conversation      []Exchange `json:"conversation"`
	AdditionalContext []string   `json:"additionalContext"`
}

type Exchange struct {
	Sender  string `json:"sender"`
	Content string `json:"content"`
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

// handleAsk handles requests for the /ask endpoint.
func handleAsk(service *Service) routing.Handler {
	return func(c *routing.Context) error {
		var rawRequest struct {
			Message string `json:"message"`
		}

		if err := c.Read(&rawRequest); err != nil {
			log.Printf("Failed to read request: %v", err)
			return c.Write(map[string]string{"response": "Invalid request format"})
		}

		var askSchema struct {
			Conversation      string   `json:"conversation"`
			AdditionalContext []string `json:"additionalContext"`
		}

		if err := json.Unmarshal([]byte(rawRequest.Message), &askSchema); err != nil {
			log.Printf("Failed to parse JSON string: %v", err)
			return c.Write(map[string]string{"response": "Invalid JSON format"})
		}

		msgPackData, err := base64.StdEncoding.DecodeString(askSchema.Conversation)
		if err != nil {
			log.Printf("Failed to decode Base64: %v", err)
			return c.Write(map[string]string{"response": "Invalid Base64 encoding"})
		}

		// First, unmarshal into an interface{} to inspect structure
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

		// var conversation []Exchange
		// if err := msgpack.Unmarshal(msgPackData, &conversation); err != nil {
		// 	log.Printf("Failed to parse MessagePack: %v", err)
		// 	return c.Write(map[string]string{"response": "Invalid MessagePack format"})
		// }

		jsonBytes, err := json.Marshal(AskSchema{
			Conversation:      conversation,
			AdditionalContext: askSchema.AdditionalContext,
		})
		if err != nil {
			return c.Write(map[string]string{"response": "Failed to marshal request"})
		}

		resp, err := service.SendMessage(context.Background(), string(jsonBytes))
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		parsedResponse, err := ParseResponse(strings.Join(resp, "\n"))
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		responseJSON, err := json.Marshal(parsedResponse)
		if err != nil {
			return c.Write(map[string]string{"response": "Failed to marshal response"})
		}
		return c.Write(map[string]interface{}{"response": string(responseJSON)})
	}
}

// handleDataRequest is a generic handler for data-object-based endpoints.
func handleDataRequest(service *Service, promptGenerator func(string) string) routing.Handler {
	return func(c *routing.Context) error {
		var request struct {
			Message string `json:"message"`
		}
		if err := c.Read(&request); err != nil {
			return c.Write(map[string]string{"response": "Invalid request"})
		}

		if request.Message == "" {
			return c.Write(map[string]string{"response": "Query is required."})
		}

		prompt := promptGenerator(request.Message)

		// Call the service to process the request.
		resp, err := service.SendMessage(context.Background(), prompt)
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		dataObjects, err := ParseResponse(strings.Join(resp, "\n"))
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		// Convert dataObjects to JSON string
		jsonStr, err := json.Marshal(dataObjects)
		if err != nil {
			return c.Write(map[string]string{"response": "Failed to serialize response"})
		}
		return c.Write(map[string]interface{}{"response": string(jsonStr)})
	}
}

// handleFind handles requests for the /find endpoint.
func handleFind(service *Service) routing.Handler {
	return handleDataRequest(service, func(message string) string {
		return fmt.Sprintf(`
		{
			"specializedTask": "-@escapeMarkup Locate contextually matching and relevant Bible (KJV) scriptures",
			"responseSchema": [
				{
					"book": "VALID_BIBLE_BOOK_NAME",
					"chapterNo": INTEGER,
					"verseNo": INTEGER,
					"verseContent": "STRING"
				}
			],
			"userProvidedContext": "%s"
		}
		`, message)
	})
}

// handleGenealogy handles requests for the /genealogy endpoint.
func handleGenealogy(service *Service) routing.Handler {
	return handleDataRequest(service, func(message string) string {
		return fmt.Sprintf(`
		{
			"specializedTask": "-@escapeMarkup Construct a genealogy tree of depth 10 generations",
			"responseSchema": {
				"personSchema": {
					"name": "string",
					"gender": "string (male or female)",
					"spouse": {
						"name": "string",
						"gender": "string (male or female)"
					},
					"parents": "array of personSchema objects of depth 10 generations"
				}
			},
			"personOfInterest": "%s"
		}
		`, message)
	})
}
