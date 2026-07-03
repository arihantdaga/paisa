package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	response string
	err      error
}

func (m mockProvider) Complete(_ context.Context, _, _ string) (string, error) {
	return m.response, m.err
}

var chart = []string{"Expenses:Food", "Expenses:Shopping", "Income:Salary", "Assets:Bank:HDFC"}

func categorizeOne(t *testing.T, response string, threshold float64) Result {
	t.Helper()
	txns := []Transaction{{ID: "0", Description: "x", KnownAccount: "Assets:Bank:HDFC"}}
	results, err := Categorize(context.Background(), mockProvider{response: response}, txns, chart, "", threshold)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	return results[0]
}

func TestCategorizeConfident(t *testing.T) {
	r := categorizeOne(t, `{"transactions":[{"id":"0","payee":"Swiggy","account":"Expenses:Food","confidence":0.95,"alternatives":[{"account":"Expenses:Shopping","confidence":0.2}]}]}`, 0.6)
	assert.Equal(t, "Expenses:Food", r.Account)
	assert.Equal(t, "Swiggy", r.Payee)
	assert.False(t, r.Flagged, "high confidence, known account, far alternative should not be flagged")
}

func TestCategorizeFlagsLowConfidence(t *testing.T) {
	r := categorizeOne(t, `{"transactions":[{"id":"0","account":"Expenses:Food","confidence":0.4,"alternatives":[]}]}`, 0.6)
	assert.True(t, r.Flagged, "confidence below threshold should flag")
}

func TestCategorizeFlagsUnknownAccount(t *testing.T) {
	r := categorizeOne(t, `{"transactions":[{"id":"0","account":"Expenses:Groceries","confidence":0.99,"alternatives":[]}]}`, 0.6)
	assert.True(t, r.Flagged, "account not in chart should flag even at high confidence")
}

func TestCategorizeFlagsCloseAlternatives(t *testing.T) {
	r := categorizeOne(t, `{"transactions":[{"id":"0","account":"Expenses:Food","confidence":0.7,"alternatives":[{"account":"Expenses:Shopping","confidence":0.62}]}]}`, 0.6)
	assert.True(t, r.Flagged, "top two within margin should flag")
}

func TestCategorizeMissingResultFallsBack(t *testing.T) {
	// model returns nothing for id "0"
	r := categorizeOne(t, `{"transactions":[]}`, 0.6)
	assert.True(t, r.Flagged, "a dropped transaction should be flagged for manual review")
	assert.Equal(t, "0", r.ID)
}

func TestCategorizeToleratesCodeFence(t *testing.T) {
	r := categorizeOne(t, "```json\n{\"transactions\":[{\"id\":\"0\",\"account\":\"Expenses:Food\",\"confidence\":0.95}]}\n```", 0.6)
	assert.Equal(t, "Expenses:Food", r.Account)
	assert.False(t, r.Flagged)
}

func TestCategorizeEmptyInput(t *testing.T) {
	results, err := Categorize(context.Background(), mockProvider{}, nil, chart, "", 0.6)
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestFormatLedgerExamples(t *testing.T) {
	ledger := `2026/01/01 Swiggy
    Expenses:Restaurants    380 INR
    Assets:Bank:HDFC

2026/02/01 PAYTM GROCERY
    Expenses:Food    1240 INR
    Assets:Bank:HDFC

2026/03/01 Swiggy
    Expenses:Restaurants    420 INR
    Assets:Bank:HDFC
`
	out := FormatLedgerExamples(ledger, 10)
	assert.Contains(t, out, "PAYTM GROCERY")
	assert.Contains(t, out, "Expenses:Food")
	// Deduped by payee: only the most recent Swiggy (03/01), not the 01/01 one.
	assert.Contains(t, out, "2026/03/01 Swiggy")
	assert.NotContains(t, out, "2026/01/01 Swiggy")
	// Two unique payees -> two blocks.
	assert.Equal(t, 2, strings.Count(out, "\n\n")+1)

	// Cap is respected.
	capped := FormatLedgerExamples(ledger, 1)
	assert.Equal(t, 1, strings.Count(capped, "\n\n")+1)
}
