package gemini

// processed structure from the /ask endpoint
type ChatSchema struct {
	Conversation []Exchange          `json:"conversation"`
	Context      []map[string]string `json:"context"`
}

// exchange struct for the AskSchema struct
type Exchange struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
