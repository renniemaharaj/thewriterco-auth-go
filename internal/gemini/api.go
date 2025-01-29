package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	routing "github.com/go-ozzo/ozzo-routing/v2"
)

// RegisterHandlers registers the routes for the GenAI API.
func RegisterHandlers(r *routing.Router, service *Service) {
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

// handleAsk handles requests for the /ask endpoint.
func handleAsk(service *Service) routing.Handler {
	return func(c *routing.Context) error {
		var request struct {
			Message string `json:"message"`
		}
		if err := c.Read(&request); err != nil {
			return c.Write(map[string]string{"error": "Invalid request"})
		}

		if request.Message == "" {
			return c.Write(map[string]string{"error": "Message is required."})
		}

		// Use the GenAI service to get a response to the message.
		resp, err := service.SendMessage(context.Background(), request.Message)
		if err != nil {
			return c.Write(map[string]string{"error": err.Error()})
		}

		fmt.Println("Responding to frontend with : ", RemoveCodeFences(strings.Join(resp, "\n")))

		return c.Write(map[string]interface{}{"response": RemoveCodeFences(strings.Join(resp, "\n"))})
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
