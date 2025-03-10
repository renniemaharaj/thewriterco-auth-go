package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	routing "github.com/go-ozzo/ozzo-routing/v2"
	"github.com/google/generative-ai-go/genai"

	"github.com/renniemaharaj/thewriterco-auth-go/pkg/pool"
	"github.com/renniemaharaj/thewriterco-auth-go/pkg/transformer"
	"github.com/renniemaharaj/thewriterco-auth-go/pkg/transformer/gemi"
)

// RegisterHandlers registers the routes for the GenAI API.
func RegisterHandlers(r *routing.RouteGroup, p *pool.Instance) {
	r.Post("/ask", handleAsk(p))
	r.Post("/find", handleFind(p))
	r.Post("/genealogy", handleGenealogy(p))

}

// handleAsk handles requests for the /ask endpoint.
func handleAsk(p *pool.Instance) routing.Handler {
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

		// log.Printf("Decoded request into struct: %v\n", chatSchema)

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
		resp, err := p.QueuedEVS(context.Background(), input, ValidateResponseSchema, 2, 1)
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		log.Println("Responding to request")
		return c.Write(map[string]interface{}{"response": string(resp)})
	}
}

// handleDataRequest is a generic handler for data-object-based endpoints.
func handleDataRequest(promptGenerator func(string) string, p *pool.Instance) routing.Handler {
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

		session, cleanup, err := p.Queue(context.Background())
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}
		defer cleanup()

		// call the service to process the request.
		resp, err := session.SendString(context.Background(), prompt)
		if err != nil {
			return c.Write(map[string]string{"response": err.Error()})
		}

		linted := transformer.LintCodeFences(&resp, "json")

		log.Println("Responding to request, with response: ", linted)
		return c.Write(map[string]interface{}{"response": linted})
	}
}

// handleFind handles requests for the /find endpoint.
func handleFind(p *pool.Instance) routing.Handler {
	return handleDataRequest(func(message string) string {
		return fmt.Sprintf(`
		-@here Please construct a scripture response block, of verses, for verses matching the query: "%s"
		`, message)
	}, p)
}

// handleGenealogy handles requests for the /genealogy endpoint.
func handleGenealogy(p *pool.Instance) routing.Handler {
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
	}, p)
}
