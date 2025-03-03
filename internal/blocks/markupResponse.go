package blocks

// markupresponse is a struct for the MarkupResponse type.
type MarkupResponse struct {
	BlockPrimitive
	MarkupContent string `json:"markupContent"`
}
