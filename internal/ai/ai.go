// Package ai provides backend, LLM assisted categorization of imported
// transactions. The frontend parses the statement and the structural template,
// then sends normalized transactions here to be categorized by an OpenAI
// compatible model.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Transaction is a single normalized transaction sent for categorization.
type Transaction struct {
	ID           string  `json:"id"`
	Date         string  `json:"date"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Description  string  `json:"description"`
	KnownAccount string  `json:"known_account"`
}

// Alternative is an alternate account suggestion with its confidence.
type Alternative struct {
	Account    string  `json:"account"`
	Confidence float64 `json:"confidence"`
}

// Result is the categorization for a single transaction.
type Result struct {
	ID           string        `json:"id"`
	Payee        string        `json:"payee"`
	Account      string        `json:"account"`
	Note         string        `json:"note"`
	Confidence   float64       `json:"confidence"`
	Alternatives []Alternative `json:"alternatives"`
	Rationale    string        `json:"rationale"`
	Flagged      bool          `json:"flagged"`
}

// Provider is the minimal LLM interface needed for categorization. Keeping it
// small lets the OpenAI compatible client be swapped (or mocked in tests) and a
// different provider shape added later.
type Provider interface {
	// Complete sends a system and user message and returns the raw text reply.
	Complete(ctx context.Context, system, user string) (string, error)
}

// alternativeMargin is the minimum confidence gap between the chosen account and
// the next best alternative. A smaller gap means the choice is ambiguous and the
// transaction is flagged for review.
const alternativeMargin = 0.15

// Categorize runs one batch of transactions through the provider and returns a
// result per input transaction (aligned by ID), with the flag rule applied. The
// caller (frontend) controls batch size.
func Categorize(ctx context.Context, provider Provider, txns []Transaction, accounts []string, examples string, threshold float64) ([]Result, error) {
	if len(txns) == 0 {
		return []Result{}, nil
	}

	content, err := provider.Complete(ctx, buildSystemPrompt(accounts, examples), buildUserPrompt(txns))
	if err != nil {
		return nil, err
	}

	parsed, err := parseResults(content)
	if err != nil {
		return nil, fmt.Errorf("could not parse model response: %w", err)
	}

	byID := make(map[string]Result, len(parsed))
	for _, r := range parsed {
		byID[r.ID] = r
	}

	accountSet := make(map[string]bool, len(accounts))
	for _, a := range accounts {
		accountSet[a] = true
	}

	results := make([]Result, len(txns))
	for i, txn := range txns {
		r, ok := byID[txn.ID]
		if !ok {
			// The model dropped this transaction; surface it for manual review
			// instead of silently losing it.
			results[i] = Result{ID: txn.ID, Payee: txn.Description, Account: "", Confidence: 0, Flagged: true}
			continue
		}
		r.ID = txn.ID
		r.Flagged = shouldFlag(r, accountSet, threshold)
		results[i] = r
	}

	return results, nil
}

// shouldFlag decides whether a categorization needs human review.
func shouldFlag(r Result, accounts map[string]bool, threshold float64) bool {
	if r.Account == "" {
		return true
	}
	if r.Confidence < threshold {
		return true
	}
	if !accounts[r.Account] {
		// A new account the model invented. Plausible, but worth confirming.
		return true
	}
	if len(r.Alternatives) > 0 && r.Confidence-r.Alternatives[0].Confidence < alternativeMargin {
		return true
	}
	return false
}

func buildSystemPrompt(accounts []string, examples string) string {
	var b strings.Builder
	b.WriteString(`You are a personal finance assistant that categorizes bank and credit card transactions into a double-entry ledger.

For each transaction you are given:
- Choose the single best "account" to categorize it under. Prefer an account from the "Available accounts" list when a reasonable match exists. Only invent a new account (following the same Type:SubType:Name convention, e.g. Expenses:Food) when none of the existing accounts fit.
- Clean the raw description into a short, human readable "payee" (e.g. "UPI/PAYTM/9823.../GROCERY" -> "Paytm Grocery"). Keep it concise.
- Optionally add a short "note" capturing any useful detail (reference numbers, counterparty). Use an empty string when there is nothing useful.
- Provide a "confidence" between 0 and 1 for the chosen account.
- Provide up to 3 "alternatives", each an object with "account" and "confidence", ordered best first. Do not repeat the chosen account.
- Provide a one sentence "rationale".

A positive amount means money coming into the known account (income); a negative amount means money leaving it (expense or transfer). Use the sign to pick income vs expense accounts.

Return ONLY a JSON object of the exact shape:
{"transactions": [{"id": "...", "payee": "...", "account": "...", "note": "...", "confidence": 0.0, "alternatives": [{"account": "...", "confidence": 0.0}], "rationale": "..."}]}

Return one result for every input transaction, preserving its "id".

Available accounts:
`)
	if len(accounts) == 0 {
		b.WriteString("(none yet)\n")
	}
	for _, a := range accounts {
		b.WriteString(a)
		b.WriteString("\n")
	}

	if strings.TrimSpace(examples) != "" {
		b.WriteString("\nHere are real, already categorized transactions from this same source. Categorize new transactions consistently with how similar ones were categorized below:\n")
		b.WriteString(examples)
		b.WriteString("\n")
	}

	return b.String()
}

var dateLineRe = regexp.MustCompile(`^\d{4}[/-]\d{2}[/-]\d{2}`)

// FormatLedgerExamples turns the text of an existing ledger file into a compact
// few-shot block: it splits the file into transactions, dedupes by payee
// (keeping the most recent occurrence), and caps the count. The result is real
// ledger entries the model can emulate.
func FormatLedgerExamples(text string, maxTransactions int) string {
	type txn struct {
		payee string
		block []string
	}

	var txns []txn
	var cur *txn
	flush := func() {
		if cur != nil {
			txns = append(txns, *cur)
			cur = nil
		}
	}

	for _, line := range strings.Split(text, "\n") {
		if dateLineRe.MatchString(line) {
			flush()
			payee := strings.TrimSpace(dateLineRe.ReplaceAllString(line, ""))
			payee = strings.TrimSpace(strings.TrimLeft(payee, "*! "))
			cur = &txn{payee: payee, block: []string{strings.TrimRight(line, " \t")}}
		} else if cur != nil {
			if strings.TrimSpace(line) == "" {
				flush()
			} else {
				cur.block = append(cur.block, strings.TrimRight(line, " \t"))
			}
		}
	}
	flush()

	// Dedupe by payee, walking from the end so the most recent entry wins.
	seen := make(map[string]bool)
	var picked []string
	for i := len(txns) - 1; i >= 0 && len(picked) < maxTransactions; i-- {
		key := strings.ToLower(txns[i].payee)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		picked = append(picked, strings.Join(txns[i].block, "\n"))
	}

	return strings.Join(picked, "\n\n")
}

func buildUserPrompt(txns []Transaction) string {
	payload, _ := json.Marshal(txns)
	return "Transactions:\n" + string(payload)
}

// parseResults extracts the transactions array from the model's JSON reply. It
// tolerates markdown code fences that some providers wrap responses in.
func parseResults(content string) ([]Result, error) {
	content = stripCodeFence(strings.TrimSpace(content))

	var wrapped struct {
		Transactions []Result `json:"transactions"`
	}
	if err := json.Unmarshal([]byte(content), &wrapped); err == nil && wrapped.Transactions != nil {
		return wrapped.Transactions, nil
	}

	// Fall back to a bare array in case the model ignored the wrapper.
	var bare []Result
	if err := json.Unmarshal([]byte(content), &bare); err != nil {
		return nil, err
	}
	return bare, nil
}

func stripCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[i+1:] // drop the language hint line (e.g. ```json)
	}
	return strings.TrimSuffix(strings.TrimSpace(s), "```")
}
