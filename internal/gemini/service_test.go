package gemini

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_service_SendMessage(t *testing.T) {

	// Register GenAI service
	genaicfg := Config{
		APIKey:           os.Getenv("GEMINI_API_KEY"),
		ModelName:        "gemini-2.0-flash-exp",
		Temperature:      1.0,
		TopK:             40,
		TopP:             0.95,
		MaxOutputTokens:  8192,
		ResponseMIMEType: "text/plain",
	}

	// logger, _ := log.NewForTest()
	s := NewService(context.Background(), genaicfg)
	response, err := s.SendMessage(context.Background(), "test")
	assert.Nil(t, err)
	assert.NotEmpty(t, response)
}
