package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	routing "github.com/go-ozzo/ozzo-routing/v2"
	"github.com/google/generative-ai-go/genai"

	"github.com/renniemaharaj/thewriterco-auth-go/pkg/pool"
	"github.com/renniemaharaj/thewriterco-auth-go/pkg/transformer/gemi"
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

		chatSchema, err := DecodeConversation(rawRequest.Message)
		if err != nil {
			log.Printf("Failed to process request: %v", err)
			return c.Write(map[string]string{"response": "Failed to process request"})
		}

		log.Printf("Decoded request into struct: %v\n", chatSchema)

		// Convert the AskSchema conversation to a list of genai.Content objects.
		len := len(chatSchema.Conversation)
		current := chatSchema.Conversation[len-1]

		history := ExchangesToHistory(chatSchema.Conversation)

		// Create the input object for the service.
		input := gemi.Input{
			Current: genai.Text(current.Content),
			History: history,
			Context: chatSchema.Context,
		}

		inputBytes, err := json.Marshal(input)
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		log.Printf("Input Constructed: %v\n", string(inputBytes))

		// Queue with with exponential backoff and validation automatic sending.
		resp, err := pool.QueuedEVS(context.Background(), input, ValidateResponseSchema, 2, 1)
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		log.Println("Responding to request")
		return c.Write(map[string]interface{}{"response": string(resp)})
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

		session, cleanup, err := pool.Queue(context.Background())
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}
		defer cleanup()

		input := gemi.Input{
			Current: genai.Text(prompt),
		}
		// call the service to process the request.
		resp, err := session.SendInput(context.Background(), input)
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		return c.Write(map[string]interface{}{"response": string(resp)})
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
