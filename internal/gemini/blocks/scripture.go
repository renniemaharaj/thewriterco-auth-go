package blocks

// scripture is a struct for the Scripture response type.
type Scripture struct {
	BlockPrimitive
	Verses []Verse `json:"verses"`
}
