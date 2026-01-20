package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vineel/vinagents-go/internal/repository"
	"github.com/vineel/vinagents-go/internal/testutil"
)

// Helper to create a clauser and return the ID
func createTestClauser(t *testing.T, env *testutil.TestEnv, token string, title string) string {
	t.Helper()
	body := map[string]interface{}{}
	if title != "" {
		body["title"] = title
	}

	w := env.Request("POST", "/api/v1/clausers", body, token)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)
	return resp["data"].(map[string]interface{})["clauserId"].(string)
}

func TestCreateClauser_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "clauser@example.com", "password123")
	require.NoError(t, err)

	body := map[string]interface{}{
		"title": "NDA Indemnification Clause",
	}

	w := env.Request("POST", "/api/v1/clausers", body, authResp.AccessToken)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	assert.Equal(t, "success", resp["status"])

	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["clauserId"])
	assert.Equal(t, "NDA Indemnification Clause", data["title"])
	assert.Nil(t, data["clauseA"])
	assert.Nil(t, data["clauseB"])
}

func TestCreateClauser_NoTitle(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "clauser2@example.com", "password123")
	require.NoError(t, err)

	w := env.Request("POST", "/api/v1/clausers", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["clauserId"])
	assert.Nil(t, data["title"])
}

func TestCreateClauser_Unauthorized(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	w := env.Request("POST", "/api/v1/clausers", nil, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetClauser_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "get@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test Clause")

	w := env.Request("GET", "/api/v1/clausers/"+clauserID, nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, clauserID, data["clauserId"])
	assert.Equal(t, "Test Clause", data["title"])
}

func TestGetClauser_NotFound(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "notfound@example.com", "password123")
	require.NoError(t, err)

	w := env.Request("GET", "/api/v1/clausers/00000000-0000-0000-0000-000000000000", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetClauser_OtherUsersClauser(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	// Create user 1 with a clauser
	authResp1, err := env.CreateTestUser(t, "user1@example.com", "password123")
	require.NoError(t, err)
	clauserID := createTestClauser(t, env, authResp1.AccessToken, "User 1 Clause")

	// Create user 2
	authResp2, err := env.CreateTestUser(t, "user2@example.com", "password123")
	require.NoError(t, err)

	// User 2 should not be able to access user 1's clauser
	w := env.Request("GET", "/api/v1/clausers/"+clauserID, nil, authResp2.AccessToken)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListClausers_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "list@example.com", "password123")
	require.NoError(t, err)

	// Create a few clausers
	for i := 0; i < 3; i++ {
		createTestClauser(t, env, authResp.AccessToken, "")
	}

	w := env.Request("GET", "/api/v1/clausers", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	clausers := data["clausers"].([]interface{})
	assert.Len(t, clausers, 3)

	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(3), pagination["total"])
}

func TestListClausers_Empty(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "empty@example.com", "password123")
	require.NoError(t, err)

	w := env.Request("GET", "/api/v1/clausers", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	// clausers might be nil or empty array
	clausers := data["clausers"]
	if clausers != nil {
		assert.Len(t, clausers.([]interface{}), 0)
	}
}

func TestListClausers_Pagination(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "paginate@example.com", "password123")
	require.NoError(t, err)

	// Create 5 clausers
	for i := 0; i < 5; i++ {
		createTestClauser(t, env, authResp.AccessToken, "")
	}

	w := env.Request("GET", "/api/v1/clausers?limit=2&offset=0", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	clausers := data["clausers"].([]interface{})
	assert.Len(t, clausers, 2)

	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(5), pagination["total"])
	assert.Equal(t, float64(2), pagination["limit"])
	assert.Equal(t, float64(0), pagination["offset"])
}

func TestDeleteClauser_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "delete@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "To Delete")

	w := env.Request("DELETE", "/api/v1/clausers/"+clauserID, nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify it's deleted
	w2 := env.Request("GET", "/api/v1/clausers/"+clauserID, nil, authResp.AccessToken)
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

func TestDeleteClauser_NotFound(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "delnf@example.com", "password123")
	require.NoError(t, err)

	w := env.Request("DELETE", "/api/v1/clausers/00000000-0000-0000-0000-000000000000", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateClauseA_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateA@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]string{
		"value": "Party A shall indemnify Party B for all claims.",
	}

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/clause-a", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Party A shall indemnify Party B for all claims.", data["clauseA"])
}

func TestUpdateClauseB_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateB@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]string{
		"value": "Liability capped at fees paid in prior 12 months.",
	}

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/clause-b", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Liability capped at fees paid in prior 12 months.", data["clauseB"])
}

func TestUpdateAgreementA_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateAgrA@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]string{
		"value": "Full text of Agreement A goes here...",
	}

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/agreement-a", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Full text of Agreement A goes here...", data["agreementA"])
}

func TestUpdateAgreementB_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateAgrB@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]string{
		"value": "Full text of Agreement B goes here...",
	}

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/agreement-b", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Full text of Agreement B goes here...", data["agreementB"])
}

func TestUpdateTitle_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateTitle@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Original Title")

	body := map[string]string{
		"value": "Updated Title",
	}

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/title", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Updated Title", data["title"])
}

func TestUpdateRepresentedParty_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateRP@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]string{
		"value": "Acme Corporation (Licensor)",
	}

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/represented-party", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Acme Corporation (Licensor)", data["representedParty"])
}

func TestUpdateDraftingApproach_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateDA@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]string{
		"value": "Aggressive - maximize protection for client",
	}

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/drafting-approach", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Aggressive - maximize protection for client", data["draftingApproach"])
}

func TestUpdatePlaybook_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updatePB@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]string{
		"value": "Standard SaaS vendor playbook v2.1",
	}

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/playbook", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Standard SaaS vendor playbook v2.1", data["playbook"])
}

func TestUpdateCounterpartyRationale_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateCR@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]string{
		"value": "Counterparty is a Fortune 500 company with strong bargaining power",
	}

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/counterparty-rationale", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Counterparty is a Fortune 500 company with strong bargaining power", data["counterpartyRationale"])
}

func TestUpdateBusinessContext_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateBC@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]string{
		"value": "Strategic partnership deal worth $5M ARR, high priority",
	}

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/business-context", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Strategic partnership deal worth $5M ARR, high priority", data["businessContext"])
}

func TestUpdateField_NotFound(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateNF@example.com", "password123")
	require.NoError(t, err)

	body := map[string]string{
		"value": "Some value",
	}

	w := env.Request("PUT", "/api/v1/clausers/00000000-0000-0000-0000-000000000000/clause-a", body, authResp.AccessToken)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateField_MissingValue(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "updateMV@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	w := env.Request("PUT", "/api/v1/clausers/"+clauserID+"/clause-a", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetScreen_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "screen@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Screen Test")

	// Update some fields including the new ones
	env.Request("PUT", "/api/v1/clausers/"+clauserID+"/clause-a", map[string]string{"value": "Clause A text"}, authResp.AccessToken)
	env.Request("PUT", "/api/v1/clausers/"+clauserID+"/clause-b", map[string]string{"value": "Clause B text"}, authResp.AccessToken)
	env.Request("PUT", "/api/v1/clausers/"+clauserID+"/represented-party", map[string]string{"value": "Test Corp"}, authResp.AccessToken)
	env.Request("PUT", "/api/v1/clausers/"+clauserID+"/drafting-approach", map[string]string{"value": "Balanced"}, authResp.AccessToken)
	env.Request("PUT", "/api/v1/clausers/"+clauserID+"/playbook", map[string]string{"value": "Standard playbook"}, authResp.AccessToken)
	env.Request("PUT", "/api/v1/clausers/"+clauserID+"/counterparty-rationale", map[string]string{"value": "Strong counterparty"}, authResp.AccessToken)
	env.Request("PUT", "/api/v1/clausers/"+clauserID+"/business-context", map[string]string{"value": "Important deal"}, authResp.AccessToken)

	w := env.Request("GET", "/api/v1/clausers/"+clauserID+"/screen", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})

	// Check clauser data
	clauser := data["clauser"].(map[string]interface{})
	assert.Equal(t, clauserID, clauser["clauserId"])
	assert.Equal(t, "Screen Test", clauser["title"])
	assert.Equal(t, "Clause A text", clauser["clauseA"])
	assert.Equal(t, "Clause B text", clauser["clauseB"])
	assert.Equal(t, "Test Corp", clauser["representedParty"])
	assert.Equal(t, "Balanced", clauser["draftingApproach"])
	assert.Equal(t, "Standard playbook", clauser["playbook"])
	assert.Equal(t, "Strong counterparty", clauser["counterpartyRationale"])
	assert.Equal(t, "Important deal", clauser["businessContext"])

	// Check outputs is empty initially
	outputs := data["outputs"].([]interface{})
	assert.Len(t, outputs, 0)

	// Check no active run
	assert.Nil(t, data["activeRun"])
}

func TestGetScreen_NotFound(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "screenNF@example.com", "password123")
	require.NoError(t, err)

	w := env.Request("GET", "/api/v1/clausers/00000000-0000-0000-0000-000000000000/screen", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRunLenses_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "lenses@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Lens Test")

	body := map[string]interface{}{
		"lenses": []string{"risks", "opportunities"},
	}

	w := env.Request("POST", "/api/v1/clausers/"+clauserID+"/run-lenses", body, authResp.AccessToken)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["runId"])
	assert.Equal(t, "pending", data["status"])
	assert.NotEmpty(t, data["pollUrl"])
}

func TestRunLenses_NoLenses(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "nolenses@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]interface{}{
		"lenses": []string{},
	}

	w := env.Request("POST", "/api/v1/clausers/"+clauserID+"/run-lenses", body, authResp.AccessToken)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRunLenses_AlreadyRunning(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "alreadyrun@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	body := map[string]interface{}{
		"lenses": []string{"risks"},
	}

	// First request should succeed
	w1 := env.Request("POST", "/api/v1/clausers/"+clauserID+"/run-lenses", body, authResp.AccessToken)
	assert.Equal(t, http.StatusAccepted, w1.Code)

	// Second request should fail (job already running)
	w2 := env.Request("POST", "/api/v1/clausers/"+clauserID+"/run-lenses", body, authResp.AccessToken)
	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestRewrite_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "rewrite@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Rewrite Test")

	body := map[string]interface{}{
		"instructions": "Create a balanced clause with mutual indemnification",
	}

	w := env.Request("POST", "/api/v1/clausers/"+clauserID+"/rewrite", body, authResp.AccessToken)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["runId"])
	assert.Equal(t, "pending", data["status"])
}

func TestRewrite_NoInstructions(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "rewriteNI@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Test")

	// Should still work without instructions
	w := env.Request("POST", "/api/v1/clausers/"+clauserID+"/rewrite", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestGetOutputs_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "outputs@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Outputs Test")

	w := env.Request("GET", "/api/v1/clausers/"+clauserID+"/outputs", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	// Initially empty
	data := resp["data"].([]interface{})
	assert.Len(t, data, 0)
}

func TestAddFavorite_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "addfav@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Fav Test")

	// Create an output with lens-based structure (as returned by Claude)
	ctx := context.Background()
	testItemID := "test-item-id-123"
	content, _ := json.Marshal(map[string]interface{}{
		"risks": []map[string]interface{}{
			{"itemId": testItemID, "priority": "high", "text": "Risk description"},
		},
	})

	output, err := repository.NewClauserOutputRepository(env.Pool).Create(ctx, repository.CreateClauserOutputInput{
		ClauserID:    clauserID,
		Ordinal:      1,
		GroupName:    "lenses",
		GroupOrdinal: 0,
		Title:        "Lens Analysis",
		Kind:         "lens_output",
		Content:      content,
	})
	require.NoError(t, err)

	body := map[string]interface{}{
		"itemId": testItemID,
	}

	w := env.Request("POST", "/api/v1/clausers/"+clauserID+"/outputs/"+output.ClauserOutputID+"/favorite", body, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "favorites", data["groupName"])

	items := data["content"].(map[string]interface{})["items"].([]interface{})
	assert.Len(t, items, 1)

	favItem := items[0].(map[string]interface{})
	assert.Equal(t, testItemID, favItem["itemId"])
	assert.Equal(t, "risks", favItem["lens"])
	assert.Equal(t, output.ClauserOutputID, favItem["sourceOutputId"])
}

func TestAddFavorite_InvalidItemId(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "badfav@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Fav Test")

	// Create an output with lens-based structure
	ctx := context.Background()
	content, _ := json.Marshal(map[string]interface{}{
		"risks": []map[string]interface{}{
			{"itemId": "real-item-id", "priority": "high", "text": "Risk description"},
		},
	})

	output, err := repository.NewClauserOutputRepository(env.Pool).Create(ctx, repository.CreateClauserOutputInput{
		ClauserID:    clauserID,
		Ordinal:      1,
		GroupName:    "lenses",
		GroupOrdinal: 0,
		Title:        "Lens Analysis",
		Kind:         "lens_output",
		Content:      content,
	})
	require.NoError(t, err)

	body := map[string]interface{}{
		"itemId": "non-existent-item-id", // Invalid itemId
	}

	w := env.Request("POST", "/api/v1/clausers/"+clauserID+"/outputs/"+output.ClauserOutputID+"/favorite", body, authResp.AccessToken)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRemoveFavorite_Success(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "rmfav@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Remove Fav Test")

	// Create an output with lens-based structure
	ctx := context.Background()
	testItemID := "remove-test-item-id"
	content, _ := json.Marshal(map[string]interface{}{
		"risks": []map[string]interface{}{
			{"itemId": testItemID, "priority": "high", "text": "Risk description"},
		},
	})

	output, err := repository.NewClauserOutputRepository(env.Pool).Create(ctx, repository.CreateClauserOutputInput{
		ClauserID:    clauserID,
		Ordinal:      1,
		GroupName:    "lenses",
		GroupOrdinal: 0,
		Title:        "Lens Analysis",
		Kind:         "lens_output",
		Content:      content,
	})
	require.NoError(t, err)

	// Add to favorites first
	addBody := map[string]interface{}{"itemId": testItemID}
	env.Request("POST", "/api/v1/clausers/"+clauserID+"/outputs/"+output.ClauserOutputID+"/favorite", addBody, authResp.AccessToken)

	// Remove from favorites
	w := env.Request("DELETE", "/api/v1/clausers/"+clauserID+"/favorites/0", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	testutil.ParseResponse(t, w, &resp)

	data := resp["data"].(map[string]interface{})
	items := data["content"].(map[string]interface{})["items"].([]interface{})
	assert.Len(t, items, 0)
}

func TestRemoveFavorite_InvalidIndex(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.ResetDatabase(t)

	authResp, err := env.CreateTestUser(t, "rmfavbad@example.com", "password123")
	require.NoError(t, err)

	clauserID := createTestClauser(t, env, authResp.AccessToken, "Remove Fav Test")

	// Try to remove with invalid index (no favorites exist)
	w := env.Request("DELETE", "/api/v1/clausers/"+clauserID+"/favorites/0", nil, authResp.AccessToken)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
