<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import _ from "lodash";
  import type { ImportTransaction } from "$lib/import_ai";

  export let transactions: ImportTransaction[] = [];
  export let allAccounts: string[] = [];
  export let model = "";
  export let categorizing = false;
  export let progress: { done: number; total: number } | null = null;

  const dispatch = createEventDispatcher();

  let container: HTMLElement;
  let showOnlyFlagged = false;
  let editing: { id: string; field: "payee" | "account" | "note" } | null = null;
  let flagCursor = -1;

  $: categorizable = transactions.filter((t) => t.categorizeIndex >= 0);
  $: anyCategorized = _.some(categorizable, (t) => t.categorized);
  $: flaggedCount = categorizable.filter((t) => t.flagged).length;
  $: okCount = categorizable.filter((t) => t.categorized && !t.flagged).length;
  $: visible = showOnlyFlagged ? transactions.filter((t) => t.flagged) : transactions;

  // Reassign to trigger Svelte reactivity after mutating a transaction in place.
  function touch() {
    transactions = transactions;
  }

  function openEditor(id: string, field: "payee" | "account" | "note") {
    editing = editing && editing.id === id && editing.field === field ? null : { id, field };
  }

  function isEditing(id: string, field: string) {
    return editing != null && editing.id === id && editing.field === field;
  }

  function setAccount(t: ImportTransaction, account: string) {
    t.account = account;
    t.flagged = false;
    touch();
  }

  function confirm(t: ImportTransaction) {
    t.flagged = false;
    editing = null;
    touch();
  }

  function useRaw(t: ImportTransaction) {
    t.payee = t.rawDescription;
    touch();
  }

  function jumpFlag(dir: 1 | -1) {
    const flagged = transactions.filter((t) => t.flagged);
    if (flagged.length === 0) return;
    flagCursor = (flagCursor + dir + flagged.length) % flagged.length;
    const el = container?.querySelector(`#txn-${flagged[flagCursor].id}`);
    el?.scrollIntoView({ behavior: "smooth", block: "center" });
  }

  function pct(n: number) {
    return `${Math.round((n || 0) * 100)}%`;
  }
</script>

<datalist id="ai-accounts">
  {#each allAccounts as account}
    <option value={account} />
  {/each}
</datalist>

<div class="ai-review">
  <div class="header box p-3 mb-3">
    <div
      class="is-flex is-align-items-center is-justify-content-space-between is-flex-wrap-wrap gap-2"
    >
      <div class="counts">
        {#if anyCategorized}
          <span class="tag is-warning is-light">⚠ {flaggedCount} to review</span>
          <span class="tag is-success is-light">✓ {okCount} ok</span>
        {:else}
          <span class="tag is-light">{categorizable.length} transactions</span>
        {/if}
        {#if model}<span class="model has-text-grey">{model}</span>{/if}
      </div>

      <div class="actions is-flex is-align-items-center gap-2">
        {#if categorizing}
          <span class="has-text-grey is-size-7">
            {progress ? `${progress.done}/${progress.total}` : ""} categorizing…
          </span>
          <button class="button is-small" on:click={() => dispatch("cancel")}>Cancel</button>
        {:else if !anyCategorized}
          <button class="button is-small is-link" on:click={() => dispatch("categorize")}>
            Categorize with AI
          </button>
        {:else}
          {#if flaggedCount > 0}
            <button
              class="button is-small is-warning"
              on:click={() => dispatch("recategorizeFlagged")}
            >
              Re-run AI on {flaggedCount} flagged
            </button>
          {/if}
          <button class="button is-small" on:click={() => dispatch("categorize")}>Re-run all</button
          >
        {/if}

        {#if flaggedCount > 0}
          <div class="buttons has-addons mb-0">
            <button class="button is-small" title="Previous flag" on:click={() => jumpFlag(-1)}
              >◂</button
            >
            <button class="button is-small" title="Next flag" on:click={() => jumpFlag(1)}>▸</button
            >
          </div>
          <label class="checkbox is-size-7">
            <input type="checkbox" bind:checked={showOnlyFlagged} /> only ⚠
          </label>
        {/if}

        <button
          class="button is-small is-success"
          disabled={categorizing}
          on:click={() => dispatch("commit")}>Commit ▸</button
        >
      </div>
    </div>

    {#if categorizing && progress}
      <progress class="progress is-link is-small mt-2" value={progress.done} max={progress.total} />
    {/if}
  </div>

  <div class="box preview" bind:this={container}>
    {#each visible as t (t.id)}
      <div class="txn" id={`txn-${t.id}`} class:flagged={t.flagged}>
        <div class="line">
          <span class="gutter">{t.flagged ? "⚠" : ""}</span>
          <span class="date">{t.date}</span>
          {#if t.categorizeIndex >= 0}
            <button
              class="pill"
              class:warn={t.flagged}
              class:ok={t.categorized && !t.flagged}
              on:click={() => openEditor(t.id, "payee")}>{t.payee || "(payee)"}</button
            >
            {#if t.categorized && t.payee !== t.rawDescription}
              <span class="raw" title={t.rawDescription}>raw: {t.rawDescription}</span>
            {/if}
          {:else}
            <span>{t.payee}</span>
          {/if}
        </div>

        {#if isEditing(t.id, "payee")}
          <div class="popover">
            <input class="input is-small" bind:value={t.payee} on:input={touch} />
            <div class="mt-1">
              <button class="button is-small" on:click={() => useRaw(t)}>Use raw</button>
              <button class="button is-small is-light" on:click={() => (editing = null)}
                >Done</button
              >
            </div>
          </div>
        {/if}

        {#each t.postings as p, i}
          <div class="line posting">
            {#if i === t.categorizeIndex}
              <button
                class="pill account"
                class:warn={t.flagged}
                class:ok={t.categorized && !t.flagged}
                on:click={() => openEditor(t.id, "account")}>{t.account || "Uncategorized"}</button
              >
              {#if t.ai}<span class="conf">{pct(t.ai.confidence)}</span>{/if}
              <span class="amount">{p.amount}</span>
            {:else}
              <span class="known">{p.account}</span>
              <span class="amount">{p.amount}</span>
            {/if}
          </div>

          {#if i === t.categorizeIndex && isEditing(t.id, "account")}
            <div class="popover">
              <input
                class="input is-small"
                list="ai-accounts"
                bind:value={t.account}
                on:input={() => {
                  t.flagged = false;
                  touch();
                }}
              />
              {#if t.ai && t.ai.alternatives && t.ai.alternatives.length}
                <div class="alts mt-1">
                  {#each t.ai.alternatives as alt}
                    <button
                      class="tag is-link is-light"
                      on:click={() => setAccount(t, alt.account)}
                    >
                      {alt.account} · {pct(alt.confidence)}
                    </button>
                  {/each}
                </div>
              {/if}
              {#if t.ai && t.ai.rationale}
                <p class="why has-text-grey is-size-7 mt-1">{t.ai.rationale}</p>
              {/if}
              <div class="mt-1">
                <button class="button is-small is-success" on:click={() => confirm(t)}
                  >Confirm</button
                >
                <button class="button is-small is-light" on:click={() => (editing = null)}
                  >Done</button
                >
              </div>
            </div>
          {/if}
        {/each}

        {#if t.categorizeIndex >= 0 && (t.note || isEditing(t.id, "note"))}
          <div class="line note">
            <button class="pill note-pill" on:click={() => openEditor(t.id, "note")}
              >; {t.note || "(add note)"}</button
            >
          </div>
          {#if isEditing(t.id, "note")}
            <div class="popover">
              <input
                class="input is-small"
                placeholder="note"
                bind:value={t.note}
                on:input={touch}
              />
              <div class="mt-1">
                <button class="button is-small is-light" on:click={() => (editing = null)}
                  >Done</button
                >
              </div>
            </div>
          {/if}
        {:else if t.categorizeIndex >= 0 && t.categorized}
          <div class="line note">
            <button class="pill add-note" on:click={() => openEditor(t.id, "note")}>+ note</button>
          </div>
        {/if}
      </div>
    {/each}
  </div>
</div>

<style lang="scss">
  .gap-2 {
    gap: 0.5rem;
  }
  .ai-review {
    .header {
      position: sticky;
      top: 0;
      z-index: 3;
    }
    .model {
      margin-left: 0.5rem;
      font-size: 0.75rem;
    }
    .preview {
      font-family: monospace;
      font-size: 0.8rem;
      overflow-y: auto;
      max-height: calc(100vh - 280px);
    }
    .txn {
      padding: 0.25rem 0.25rem 0.25rem 0;
      border-left: 3px solid transparent;
      &.flagged {
        border-left-color: hsl(48, 100%, 67%);
        background: hsla(48, 100%, 67%, 0.08);
      }
    }
    .line {
      white-space: pre-wrap;
      line-height: 1.6;
    }
    .posting,
    .note {
      padding-left: 2rem;
    }
    .gutter {
      display: inline-block;
      width: 1rem;
      color: hsl(48, 100%, 40%);
    }
    .date {
      margin-right: 0.5rem;
    }
    .known {
      margin-right: 0.5rem;
    }
    .conf,
    .amount {
      margin-left: 0.75rem;
      color: hsl(0, 0%, 50%);
    }
    .raw {
      margin-left: 0.5rem;
      color: hsl(0, 0%, 60%);
      font-size: 0.72rem;
    }
    .pill {
      border: 1px solid hsl(0, 0%, 80%);
      border-radius: 4px;
      padding: 0 0.4rem;
      background: hsl(0, 0%, 96%);
      cursor: pointer;
      font-family: inherit;
      font-size: inherit;
      &.warn {
        border-color: hsl(48, 100%, 50%);
        background: hsla(48, 100%, 67%, 0.25);
      }
      &.ok {
        border-color: hsl(141, 53%, 53%);
        background: hsla(141, 71%, 48%, 0.12);
      }
      &.note-pill,
      &.add-note {
        color: hsl(0, 0%, 45%);
        background: transparent;
        border-style: dashed;
      }
    }
    .popover {
      margin: 0.25rem 0 0.5rem 2rem;
      padding: 0.5rem;
      border: 1px solid hsl(0, 0%, 85%);
      border-radius: 6px;
      background: hsl(0, 0%, 99%);
      max-width: 28rem;
    }
    .alts {
      display: flex;
      flex-wrap: wrap;
      gap: 0.25rem;
      .tag {
        cursor: pointer;
      }
    }
    .why {
      margin: 0;
    }
  }
</style>
