package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	routing "github.com/go-ozzo/ozzo-routing/v2"
	"github.com/renniemaharaj/thewriterco-auth-go/pkg/transformer"
)

// RegisterHandlers registers the routes for the GenAI API.
func RegisterHandlers(r *routing.RouteGroup) {
	r.Post("/ask", handleAsk())
	r.Post("/find", handleFind())
	r.Post("/genealogy", handleGenealogy())

}

// handleAsk handles requests for the /ask endpoint.
func handleAsk() routing.Handler {
	return func(c *routing.Context) error {
		var rawRequest struct {
			Message string `json:"message"`
		}

		if err := c.Read(&rawRequest); err != nil {
			log.Printf("Failed to read request: %v", err)
			return c.Write(map[string]string{"response": "Invalid request format"})
		}

		var askSchema transformer.RawAskSchema

		if err := json.Unmarshal([]byte(rawRequest.Message), &askSchema); err != nil {
			log.Printf("Failed to parse JSON string: %v", err)
			return c.Write(map[string]string{"response": "Invalid JSON format"})
		}

		treatedSchema, err := RawToAskSchema(&askSchema)

		if err != nil {
			log.Printf("Failed to process AskSchema: %v", err)
			return c.Write(map[string]string{"response": "Failed to process request"})
		}

		session, cleanup, err := WaitFreeSession(context.Background())
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}
		defer cleanup()

		// call the service to process the request.
		resp, err := session.Ask(context.Background(), treatedSchema)

		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		parsedResponse, err := ParseResponse(*resp)
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		responseJSON, err := json.Marshal(parsedResponse)
		if err != nil {
			return c.Write(map[string]string{"response": "Failed to marshal response"})
		}

		log.Printf("Responding to request: %s", responseJSON)

		return c.Write(map[string]interface{}{"response": string(responseJSON)})
	}
}

// handleDataRequest is a generic handler for data-object-based endpoints.
func handleDataRequest(promptGenerator func(string) string) routing.Handler {
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

		session, cleanup, err := WaitFreeSession(context.Background())
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}
		defer cleanup()

		// call the service to process the request.
		resp, err := session.SendMessage(context.Background(), &prompt)
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		dataObjects, err := ParseResponse(*resp)
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		jsonStr, err := json.Marshal(dataObjects)
		if err != nil {
			return c.Write(map[string]string{"response": "Failed to serialize response"})
		}
		return c.Write(map[string]interface{}{"response": string(jsonStr)})
	}
}

// handleFind handles requests for the /find endpoint.
func handleFind() routing.Handler {
	return handleDataRequest(func(message string) string {
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
func handleGenealogy() routing.Handler {
	return handleDataRequest(func(message string) string {
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
