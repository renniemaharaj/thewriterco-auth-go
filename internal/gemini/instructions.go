package gemini

import (
	// "context"
	"io"
	"os"
	// "github.com/renniemaharaj/thewriterco-auth-go/pkg/log"
)

func GetAxiomsAndInstructions() string {
	// logger := log.New().With(context.TODO(), "version")
	instrFile, err := os.Open("instructions.txt")
	if err != nil {
		// logger.Error("failed to open axiom instructions file")
		os.Exit(-1)
	}
	defer instrFile.Close()

	instrBytes, err := io.ReadAll(instrFile)
	if err != nil {
		// logger.Error("failed to read axiom instructions")
		os.Exit(-1)
	}
	return string(instrBytes)
}
