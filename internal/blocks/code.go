package blocks

// code is a struct for the Code response type.
type Code struct {
	BlockPrimitive
	Language     string `json:"language"`
	Filename     string `json:"filename"`
	CodeContent  string `json:"codeContent"`
	EditorHeight int    `json:"editorHeight,omitempty"`
}
