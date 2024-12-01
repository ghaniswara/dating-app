package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/ghaniswara/dating-app/internal/entity"
	helper_test "github.com/ghaniswara/dating-app/test/helper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Set up the test server
	resources, err := helper_test.SetupTestServer(context.TODO())
	var code int

	if err != nil {
		log.Printf("Failed to set up test server: %s", err)
		code = 1
	} else {
		// Run tests
		code = m.Run()

		// Clean up resources
	}

	resources.CleanupTestServer()
	os.Exit(code)
}

func TestSignUp(t *testing.T) {
	reqBody := entity.CreateUserRequest{
		Name:     "testname",
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}
	body, _ := json.Marshal(reqBody)

	// Create a new HTTP client
	client := &http.Client{}

	// Make a normal HTTP request
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/auth/sign-up", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Assert the response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// You may need to read the response body to check for the success message
}

func TestSignIn(t *testing.T) {
	reqBody := entity.SignInRequest{
		Email:    "asd@asd.com",
		Username: "testuser",
		Password: "password123",
	}

	_, err := helper_test.SignUpUser(reqBody.Username, reqBody.Password, reqBody.Email)

	if err != nil {
		t.Fatalf("Failed to Sign Up: %v", err)
	}

	body, _ := json.Marshal(reqBody)

	// Create a new HTTP client
	client := &http.Client{}

	// Make a request to the correct endpoint
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/auth/sign-in", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Assert the response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}