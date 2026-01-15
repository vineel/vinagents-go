package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vineel/vinagents-go/internal/testutil"
)

func TestLaunchRun_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	// Create a user
	authResp, err := env.CreateTestUser(t, "agent@example.com", "password123")
	require.NoError(t, err)

	// Launch a run
	body := map[string]interface{}{
		"input": map[string]string{
			"prompt": "Hello, world!",
		},
	}

	w := env.Request("POST", "/api/v1/agents/simple/run", body, authResp.AccessToken)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	assert.Equal(t, "success", resp["status"])

	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["runId"])
	assert.Equal(t, "pending", data["status"])
	assert.NotEmpty(t, data["pollUrl"])
}

func TestLaunchRun_UnknownAgentType(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "agent2@example.com", "password123")
	require.NoError(t, err)

	body := map[string]interface{}{
		"input": map[string]string{
			"prompt": "Hello",
		},
	}

	w := env.Request("POST", "/api/v1/agents/unknown/run", body, authResp.AccessToken)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLaunchRun_Unauthorized(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	body := map[string]interface{}{
		"input": map[string]string{
			"prompt": "Hello",
		},
	}

	w := env.Request("POST", "/api/v1/agents/simple/run", body, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListRuns_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "list@example.com", "password123")
	require.NoError(t, err)

	// Launch a few runs
	for i := 0; i < 3; i++ {
		body := map[string]interface{}{
			"input": map[string]string{
				"prompt": "Test prompt",
			},
		}
		w := env.Request("POST", "/api/v1/agents/simple/run", body, authResp.AccessToken)
		assert.Equal(t, http.StatusAccepted, w.Code)
	}

	// List runs
	w := env.Request("GET", "/api/v1/agents/runs", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	runs := data["runs"].([]interface{})
	assert.Len(t, runs, 3)

	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(3), pagination["total"])
}

func TestGetRunStatus_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "status@example.com", "password123")
	require.NoError(t, err)

	// Launch a run
	launchBody := map[string]interface{}{
		"input": map[string]string{
			"prompt": "Hello",
		},
	}
	launchResp := env.Request("POST", "/api/v1/agents/simple/run", launchBody, authResp.AccessToken)
	require.Equal(t, http.StatusAccepted, launchResp.Code)

	var launchData map[string]interface{}
	testutil.ParseResponse(t, launchResp, &launchData)
	runID := launchData["data"].(map[string]interface{})["runId"].(string)

	// Get status
	w := env.Request("GET", "/api/v1/agents/runs/"+runID, nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, runID, data["runId"])
	assert.Equal(t, "simple", data["agentType"])
}

func TestGetRunStatus_NotFound(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "notfound@example.com", "password123")
	require.NoError(t, err)

	w := env.Request("GET", "/api/v1/agents/runs/00000000-0000-0000-0000-000000000000", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCancelRun_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "cancel@example.com", "password123")
	require.NoError(t, err)

	// Launch a run
	launchBody := map[string]interface{}{
		"input": map[string]string{
			"prompt": "Hello",
		},
	}
	launchResp := env.Request("POST", "/api/v1/agents/simple/run", launchBody, authResp.AccessToken)
	require.Equal(t, http.StatusAccepted, launchResp.Code)

	var launchData map[string]interface{}
	testutil.ParseResponse(t, launchResp, &launchData)
	runID := launchData["data"].(map[string]interface{})["runId"].(string)

	// Cancel the run
	w := env.Request("POST", "/api/v1/agents/runs/"+runID+"/cancel", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "cancel_requested", data["status"])
}

func TestListRuns_WithStatusFilter(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "filter@example.com", "password123")
	require.NoError(t, err)

	// Launch a run
	body := map[string]interface{}{
		"input": map[string]string{
			"prompt": "Test",
		},
	}
	env.Request("POST", "/api/v1/agents/simple/run", body, authResp.AccessToken)

	// List only pending runs
	w := env.Request("GET", "/api/v1/agents/runs?status=pending", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	runs := data["runs"].([]interface{})
	assert.Len(t, runs, 1)
}
