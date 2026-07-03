import _ from "lodash";
import { format } from "$lib/journal";
import { renderEach } from "$lib/spreadsheet";
import { CATEGORIZE_ACCOUNT } from "$lib/template_helpers";
import { ajax } from "$lib/utils";

export interface ParsedPosting {
  account: string;
  amount: string; // raw amount string as rendered, e.g. "-1240 INR" or ""
}

export interface AICategorization {
  payee: string;
  account: string;
  note: string;
  confidence: number;
  alternatives: { account: string; confidence: number }[];
  rationale: string;
  flagged: boolean;
}

// ImportTransaction holds both the parsed (template) output and the current
// working state edited by the user / filled by the AI.
export interface ImportTransaction {
  id: string;
  sourceRowIndex: number;
  date: string;
  rawDescription: string; // payee as produced by the template, before AI cleanup
  postings: ParsedPosting[];
  categorizeIndex: number; // index of the {{categorize}} posting, -1 if none
  knownAccount: string;
  amount: number; // signed delta from the known account's perspective (+ in, - out)
  currency: string;

  // working / editable state
  payee: string;
  account: string;
  note: string;
  flagged: boolean;
  categorized: boolean; // true once the AI has returned a result for this txn
  ai?: AICategorization;
}

const DATE_LINE = /^(\d{4}[/-]\d{2}[/-]\d{2})\s*(.*)$/;

function parsePosting(line: string): ParsedPosting {
  const trimmed = line.trim();
  const sep = trimmed.search(/\s{2,}|\t/);
  if (sep === -1) {
    return { account: trimmed, amount: "" };
  }
  return { account: trimmed.slice(0, sep).trim(), amount: trimmed.slice(sep).trim() };
}

function amountNumber(amount: string): number {
  const match = amount.replace(/,/g, "").match(/-?\d+(\.\d+)?/);
  return match ? parseFloat(match[0]) : NaN;
}

function amountCurrency(amount: string): string {
  return amount
    .replace(/-?[\d.,]+/g, "")
    .replace(/[()]/g, "")
    .trim();
}

function parseTransaction(text: string, index: number): ImportTransaction | null {
  const lines = text.split("\n");
  const first = (lines[0] || "").trim();
  const dateMatch = first.match(DATE_LINE);
  if (!dateMatch) {
    return null; // not a transaction block (e.g. a stray price or comment)
  }

  const postings: ParsedPosting[] = [];
  for (const line of lines.slice(1)) {
    const t = line.trim();
    if (!t || t.startsWith(";")) {
      continue;
    }
    postings.push(parsePosting(t));
  }

  const categorizeIndex = postings.findIndex((p) => p.account === CATEGORIZE_ACCOUNT);
  const knownPosting = postings.find((_p, i) => i !== categorizeIndex);
  const knownAccount = knownPosting ? knownPosting.account : "";

  // Amount relative to the known account: the categorize posting carries the
  // amount on the opposite leg, so negate it. Fall back to the known leg.
  let amount = NaN;
  let currency = "";
  if (categorizeIndex >= 0 && postings[categorizeIndex].amount) {
    amount = -amountNumber(postings[categorizeIndex].amount);
    currency = amountCurrency(postings[categorizeIndex].amount);
  } else if (knownPosting && knownPosting.amount) {
    amount = amountNumber(knownPosting.amount);
    currency = amountCurrency(knownPosting.amount);
  }

  return {
    id: String(index),
    sourceRowIndex: index,
    date: dateMatch[1],
    rawDescription: dateMatch[2].trim(),
    postings,
    categorizeIndex,
    knownAccount,
    amount: isNaN(amount) ? 0 : amount,
    currency,
    payee: dateMatch[2].trim(),
    account: categorizeIndex >= 0 ? "" : postings[0]?.account ?? "",
    note: "",
    flagged: false,
    categorized: false
  };
}

// parseRenderedTransactions renders the template per row and parses each result
// into a structured transaction.
export function parseRenderedTransactions(
  rows: Array<Record<string, any>>,
  template: Handlebars.TemplateDelegate
): ImportTransaction[] {
  return _.chain(renderEach(rows, template))
    .map(({ index, text }) => parseTransaction(text, index))
    .compact()
    .value();
}

// hasCategorizeMarker reports whether the template opts into backend AI
// categorization (uses the {{categorize}} helper).
export function hasCategorizeMarker(transactions: ImportTransaction[]): boolean {
  return _.some(transactions, (t) => t.categorizeIndex >= 0);
}

function applyResult(txn: ImportTransaction, result: AICategorization) {
  txn.ai = result;
  txn.payee = result.payee || txn.rawDescription;
  txn.account = result.account || txn.account;
  txn.note = result.note || "";
  txn.flagged = result.flagged;
  txn.categorized = true;
}

interface CategorizeOptions {
  batchSize: number;
  exampleLedger?: string;
  signal?: AbortSignal;
  onProgress?: (done: number, total: number) => void;
}

// categorizeBatches sends the transactions to the backend in batches, mutating
// each transaction in place with its AI result as batches complete. The caller
// triggers reactivity (e.g. reassigns the array) inside onProgress. Throws on a
// backend error; respects opts.signal for cancellation.
export async function categorizeBatches(
  transactions: ImportTransaction[],
  opts: CategorizeOptions
): Promise<void> {
  const targets = transactions.filter((t) => t.categorizeIndex >= 0);
  const byId = _.keyBy(targets, "id");
  const batches = _.chunk(targets, Math.max(1, opts.batchSize || 10));

  let done = 0;
  for (const batch of batches) {
    if (opts.signal?.aborted) {
      return;
    }

    const response = await ajax("/api/import/categorize", {
      method: "POST",
      background: true,
      signal: opts.signal,
      body: JSON.stringify({
        example_ledger: opts.exampleLedger || "",
        transactions: batch.map((t) => ({
          id: t.id,
          date: t.date,
          amount: t.amount,
          currency: t.currency,
          description: t.rawDescription,
          known_account: t.knownAccount
        }))
      })
    });

    if (response?.error) {
      throw new Error(response.error);
    }

    for (const result of response.results || []) {
      const txn = byId[result.id];
      if (txn) {
        applyResult(txn, result);
      }
    }

    done += batch.length;
    opts.onProgress?.(done, targets.length);
  }
}

// toJournal serializes the confirmed transactions back to ledger text using the
// current (possibly user-edited) working state.
export function toJournal(transactions: ImportTransaction[]): string {
  const blocks = transactions.map((t) => {
    const lines = [`${t.date} ${t.payee}`.trimEnd()];
    if (t.note && t.note.trim()) {
      lines.push(`    ; ${t.note.trim()}`);
    }
    t.postings.forEach((p, i) => {
      const account = i === t.categorizeIndex ? t.account || p.account : p.account;
      lines.push(p.amount ? `    ${account}        ${p.amount}` : `    ${account}`);
    });
    return lines.join("\n");
  });
  return format(blocks.join("\n\n"));
}
