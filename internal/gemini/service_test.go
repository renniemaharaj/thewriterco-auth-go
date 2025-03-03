package gemini

import (
	"context"
	"testing"

	"github.com/renniemaharaj/thewriterco-auth-go/pkg/transformer"
	"github.com/stretchr/testify/assert"
)

func Test_service_SendMessage(t *testing.T) {

	// logger, _ := log.NewForTest()
	session, cleanup, err := WaitFreeSession(context.Background())
	if err != nil {
		t.Error(err)
	}
	defer cleanup()

	response, err := session.Ask(context.Background(), &transformer.AskSchema{
		Conversation: []transformer.Exchange{
			{
				Role:    "user",
				Content: "This is a test message",
			},
		},
		AdditionalContext: []string{
			"key",
		},
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, response)
}
