package transformer

import (
	"io"
	"os"
)

// GetProgramming returns the system instructions
func GetProgramming() string {

	instrFile, err := os.Open("instructions.txt")
	if err != nil {
		os.Exit(-1)
	}

	defer instrFile.Close()

	instrBytes, err := io.ReadAll(instrFile)
	if err != nil {
		os.Exit(-1)
	}
	return string(instrBytes)
}
