package auth

import (
	"context"
	"net/http"
	"testing"

	"github.com/renniemaharaj/thewriterco-auth-go/internal/errors"
	"github.com/renniemaharaj/thewriterco-auth-go/internal/test"
	"github.com/renniemaharaj/thewriterco-auth-go/pkg/log"
)

type mockService struct{}

func (m mockService) Login(ctx context.Context, username, password string) (string, error) {
	if username == "test" && password == "pass" {
		return "token-100", nil
	}
	return "", errors.Unauthorized("")
}

func TestAPI(t *testing.T) {
	logger, _ := log.NewForTest()
	router := test.MockRouter(logger)
	RegisterHandlers(router.Group(""), mockService{}, logger)

	tests := []test.APITestCase{
		{"success", "POST", "/login", `{"username":"test","password":"pass"}`, nil, http.StatusOK, `{"token":"token-100"}`},
		{"bad credential", "POST", "/login", `{"username":"test","password":"wrong pass"}`, nil, http.StatusUnauthorized, ""},
		{"bad json", "POST", "/login", `"username":"test","password":"wrong pass"}`, nil, http.StatusBadRequest, ""},
	}
	for _, tc := range tests {
		test.Endpoint(t, router, tc)
	}
}
