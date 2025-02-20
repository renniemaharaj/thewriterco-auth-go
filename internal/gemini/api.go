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

// RemoveCodeFences removes ```json from the start and ``` from the end of the input string.
func RemoveCodeFences(input string) string {
	const codeFenceStart = "```json"
	const codeFenceEnd = "```"

	// trim the starting "```json"
	input = strings.TrimPrefix(input, codeFenceStart)

	// trim any leading/trailing whitespace or newlines to better detect the ending code fence
	input = strings.TrimSpace(input)

	// trim the ending "```"
	input = strings.TrimSuffix(input, codeFenceEnd)

	// trim excess whitespace again
	return strings.TrimSpace(input)
}

// function to parse JSON into the ResponseSchema struct
func ParseResponse(jsonData string) (interface{}, error) {
	// create an empty ResponseSchema instance
	var response interface{}

	// unmarshal the input JSON into the struct
	err := json.Unmarshal([]byte(RemoveCodeFences(jsonData)), &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func RawToAskSchema(raw *RawAskSchema) (*AskSchema, error) {
	// return
	msgPackData, err := base64.StdEncoding.DecodeString(raw.Conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Base64: %w", err)
	}

	var intermediate interface{}
	if err := msgpack.Unmarshal(msgPackData, &intermediate); err != nil {
		return nil, fmt.Errorf("failed to parse MessagePack (raw output): %w", err)
	}

	// Ensure correct type conversion
	var conversation []Exchange
	arr, ok := intermediate.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected MessagePack structure: %T", intermediate)
	}

	conversation = make([]Exchange, len(arr))
	for i, v := range arr {
		if obj, ok := v.(map[string]interface{}); ok {
			conversation[i] = Exchange{
				Sender:  obj["sender"].(string),
				Content: obj["content"].(string),
			}
		}
	}

	return &AskSchema{
		Conversation:      conversation,
		AdditionalContext: raw.AdditionalContext,
	}, nil
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

		var askSchema RawAskSchema

		if err := json.Unmarshal([]byte(rawRequest.Message), &askSchema); err != nil {
			log.Printf("Failed to parse JSON string: %v", err)
			return c.Write(map[string]string{"response": "Invalid JSON format"})
		}

		// convert the RawAskSchema to AskSchema
		treatedSchema, err := RawToAskSchema(&askSchema)

		if err != nil {
			log.Printf("Failed to process AskSchema: %v", err)
			return c.Write(map[string]string{"response": "Failed to process request"})
		}

		// marshal the AskSchema struct into bytes
		jsonBytes, err := json.Marshal(treatedSchema)

		if err != nil {
			return c.Write(map[string]string{"response": "Failed to marshal request"})
		}

		// call the service with jsonBytes string cast to process the request
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

		// call the service to process the request.
		resp, err := service.SendMessage(context.Background(), prompt)
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		dataObjects, err := ParseResponse(strings.Join(resp, "\n"))
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		// convert dataObjects to JSON string
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
			"specializedTask": "-@escapeDefaultResponseSchema Locate contextually matching and relevant Bible (KJV) scriptures",
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
			"specializedTask": "-@escapeDefaultResponseSchema Construct a genealogy tree of depth 10 generations",
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
