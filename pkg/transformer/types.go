package transformer

// expected structure for the /ask endpoint
type RawAskSchema struct {
	Conversation      string   `json:"conversation"`
	AdditionalContext []string `json:"additionalContext"`
}

// processed structure from the /ask endpoint
type AskSchema struct {
	Conversation      []Exchange `json:"conversation"`
	AdditionalContext []string   `json:"additionalContext"`
}

// exchange struct for the AskSchema struct
type Exchange struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
