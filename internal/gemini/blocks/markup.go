package blocks

// markup is a struct for the Markup type.
type Markup struct {
	BlockPrimitive
	MarkupContent string `json:"markupContent"`
}
