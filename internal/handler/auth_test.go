package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vineel/vinagents-go/internal/testutil"
)

func TestHealthEndpoint(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	w := env.Request("GET", "/api/v1/health", nil, "")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	assert.Equal(t, "success", resp["status"])
	assert.Equal(t, "API is running", resp["message"])
}

func TestRegister_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}

	w := env.Request("POST", "/api/v1/auth/register", body, "")

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	assert.Equal(t, "success", resp["status"])

	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["accessToken"])
	assert.NotEmpty(t, data["refreshToken"])

	user := data["user"].(map[string]interface{})
	assert.Equal(t, "test@example.com", user["email"])
}

func TestRegister_DuplicateEmail(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	body := map[string]string{
		"email":    "duplicate@example.com",
		"password": "password123",
	}

	// First registration should succeed
	w1 := env.Request("POST", "/api/v1/auth/register", body, "")
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Second registration with same email should fail
	w2 := env.Request("POST", "/api/v1/auth/register", body, "")
	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestLogin_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	// First register a user
	_, err := env.CreateTestUser(t, "login@example.com", "password123")
	require.NoError(t, err)

	// Then login
	body := map[string]string{
		"email":    "login@example.com",
		"password": "password123",
	}

	w := env.Request("POST", "/api/v1/auth/login", body, "")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	assert.Equal(t, "success", resp["status"])

	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["accessToken"])
	assert.NotEmpty(t, data["refreshToken"])
}

func TestLogin_WrongPassword(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	// First register a user
	_, err := env.CreateTestUser(t, "wrongpw@example.com", "correctpassword")
	require.NoError(t, err)

	// Try to login with wrong password
	body := map[string]string{
		"email":    "wrongpw@example.com",
		"password": "wrongpassword",
	}

	w := env.Request("POST", "/api/v1/auth/login", body, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_UserNotFound(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	body := map[string]string{
		"email":    "notfound@example.com",
		"password": "password123",
	}

	w := env.Request("POST", "/api/v1/auth/login", body, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetMe_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	// Register a user and get token
	authResp, err := env.CreateTestUser(t, "me@example.com", "password123")
	require.NoError(t, err)

	// Get current user
	w := env.Request("GET", "/api/v1/users/me", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "me@example.com", data["email"])
}

func TestGetMe_Unauthorized(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	w := env.Request("GET", "/api/v1/users/me", nil, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetMe_InvalidToken(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	w := env.Request("GET", "/api/v1/users/me", nil, "invalid-token")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
