package blocks

// verse is a struct for to be used in the Scripture struct.
type Verse struct {
	Book         string `json:"book"`
	ChapterNo    int    `json:"chapterNo"`
	VerseNo      int    `json:"verseNo"`
	VerseContent string `json:"verseContent"`
}
