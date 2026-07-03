package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOpenAIProviderEndToEnd exercises the real HTTP client path against a mock
// OpenAI compatible server and feeds the reply through Categorize.
func TestOpenAIProviderEndToEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "gpt-test")
		assert.Contains(t, string(body), "Available accounts")

		reply := `{"transactions":[{"id":"0","payee":"Swiggy","account":"Expenses:Food","note":"","confidence":0.92,"alternatives":[{"account":"Expenses:Shopping","confidence":0.1}],"rationale":"food delivery"}]}`
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": reply}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider(server.URL, "test-key", "gpt-test")
	assert.NoError(t, err)

	txns := []Transaction{{ID: "0", Date: "2026/03/14", Amount: -380, Currency: "INR", Description: "SWIGGY", KnownAccount: "Assets:Bank:HDFC"}}
	results, err := Categorize(context.Background(), provider, txns, chart, "", 0.6)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Expenses:Food", results[0].Account)
	assert.Equal(t, "Swiggy", results[0].Payee)
	assert.False(t, results[0].Flagged)
}

func TestOpenAIProviderPropagatesErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer server.Close()

	provider, _ := NewOpenAIProvider(server.URL, "bad-key", "gpt-test")
	_, err := provider.Complete(context.Background(), "sys", "user")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "invalid api key"))
}

func TestNewOpenAIProviderValidates(t *testing.T) {
	_, err := NewOpenAIProvider("", "k", "m")
	assert.Error(t, err, "missing base_url should error")
	_, err = NewOpenAIProvider("http://x", "", "m")
	assert.Error(t, err, "missing api_key should error")
	_, err = NewOpenAIProvider("http://x", "k", "")
	assert.Error(t, err, "missing model should error")
}
