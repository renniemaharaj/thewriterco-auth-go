package gemini

// responseblockprimitive is a common struct for all response types.
type ResponseBlockPrimitive struct {
	Type string `json:"type"`
}

// markupresponse is a struct for the MarkupResponse type.
type MarkupResponse struct {
	ResponseBlockPrimitive
	MarkupContent string `json:"markupContent"`
}

// verse is a struct for to be used in the Scripture struct.
type Verse struct {
	Book         string `json:"book"`
	ChapterNo    int    `json:"chapterNo"`
	VerseNo      int    `json:"verseNo"`
	VerseContent string `json:"verseContent"`
}

// scripture is a struct for the Scripture response type.
type Scripture struct {
	ResponseBlockPrimitive
	Verses []Verse `json:"verses"`
}

// code is a struct for the Code response type.
type Code struct {
	ResponseBlockPrimitive
	Language     string `json:"language"`
	Filename     string `json:"filename"`
	CodeContent  string `json:"codeContent"`
	EditorHeight int    `json:"editorHeight,omitempty"`
}

// standard Response Block Structure
type ResponseBlock struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"` // Could be MarkupResponse, Scripture, or Code
}

// default Response Schema
type ResponseSchema struct {
	ResponseBlocks []ResponseBlock `json:"responseBlocks"`
}

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
	Sender  string `json:"sender"`
	Content string `json:"content"`
}
