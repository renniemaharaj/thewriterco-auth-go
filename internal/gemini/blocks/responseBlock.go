package blocks

// standard Response Block Structure
type ResponseBlock struct {
	BlockPrimitive
	Content interface{} `json:"content"` // Could be MarkupResponse, Scripture, or Code
}
