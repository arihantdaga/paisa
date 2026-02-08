# Paisa Frontend Developer Guide

## Overview

The Paisa frontend is built with SvelteKit, a modern web framework that provides server-side rendering, routing, and excellent developer experience. The UI uses Bulma CSS framework with custom styling.

## Technology Stack

- **Framework**: SvelteKit 2.x with Svelte 4
- **Build Tool**: Vite
- **Styling**: 
  - Bulma CSS framework
  - Tailwind CSS utilities
  - Custom SCSS modules
- **Charts**: D3.js for visualizations
- **Tables**: Tabulator for data grids
- **Code Editor**: CodeMirror 6
- **Date Handling**: Day.js
- **Icons**: Font Awesome, Material Design Icons, custom icon fonts

## Project Structure

```
/src
  /lib                    # Shared components and utilities
    /components          # Reusable Svelte components
    /sheet              # Sheet parser and interpreter
    /search             # Search query parser
  /routes               # SvelteKit routes (file-based routing)
    /(app)              # Main application routes
      /assets           # Asset management pages
      /cash_flow        # Cash flow pages
      /expense          # Expense tracking
      /income           # Income tracking
      /ledger           # Ledger management
      /liabilities      # Liability management
      /more             # Additional features
    /login              # Authentication page
  /colors.scss          # Color definitions
  /common.scss          # Common styles
  /dark.scss           # Dark theme
  /light.scss          # Light theme
  app.d.ts             # TypeScript definitions
  app.html             # HTML template
  app.scss             # Global styles
  store.ts             # Svelte stores
```

## Key Components

### 1. Layout Structure

The app uses a nested layout structure:

```
+layout.svelte (root)
└── (app)/+layout.svelte (authenticated app layout)
    ├── Navbar.svelte
    └── [page content]
```

### 2. Reusable Components (`/src/lib/components/`)

#### Data Display
- `Table.svelte` - Wrapper for Tabulator tables
- `AssetsBalance.svelte` - Asset balance display
- `PostingCard.svelte` - Transaction posting display
- `TransactionCard.svelte` - Full transaction display

#### Form Controls
- `AccountsSelect.svelte` - Account selection dropdown
- `DateRange.svelte` - Date range picker
- `MonthPicker.svelte` - Month selection
- `SearchQuery.svelte` - Search query builder

#### UI Elements
- `Modal.svelte` - Modal dialog wrapper
- `Spinner.svelte` - Loading indicator
- `ZeroState.svelte` - Empty state display
- `BoxedTabs.svelte` - Tabbed interface

#### Charts & Visualizations
- Various D3.js based chart components
- Custom visualization components

### 3. Stores (`/src/store.ts`)

```javascript
// Persisted stores with localStorage
export const firstName = persisted("firstName", "");
export const baseCurrency = persisted("base_currency", "INR");

// Regular stores
export const currentQuery = writable("");
```

### 4. Page Structure

Each page typically follows this pattern:

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  
  let data = null;
  let loading = true;
  
  onMount(async () => {
    const response = await fetch('/api/endpoint');
    data = await response.json();
    loading = false;
  });
</script>

<section class="section">
  <div class="container">
    {#if loading}
      <Spinner />
    {:else}
      <!-- Page content -->
    {/if}
  </div>
</section>
```

## Routing

SvelteKit uses file-based routing:

- `/src/routes/+page.svelte` → `/`
- `/src/routes/assets/balance/+page.svelte` → `/assets/balance`
- `/src/routes/ledger/editor/[slug]/+page.svelte` → `/ledger/editor/:slug`

### Route Groups

The `(app)` folder groups authenticated routes that share a common layout.

## Data Fetching

### Client-Side Fetching

```javascript
import { ajax, type SearchQuery } from "$lib/utils";

const { data: transactions } = await ajax("/api/transaction");
```

### Load Functions

```javascript
// +page.ts
export async function load({ params, url }) {
  const response = await fetch(`/api/data/${params.id}`);
  const data = await response.json();
  
  return {
    data
  };
}
```

## State Management

### Local Component State

```svelte
<script lang="ts">
  let selectedAccount = "";
  let dateRange = { start: null, end: null };
</script>
```

### Global State (Stores)

```javascript
import { accountTrie } from "$lib/utils";

$: accounts = $accountTrie.accounts;
```

### Reactive Statements

```svelte
<script>
  $: filteredData = data.filter(d => d.account === selectedAccount);
  $: total = filteredData.reduce((sum, d) => sum + d.amount, 0);
</script>
```

## Working with Tables

### Tabulator Integration

```javascript
import { renderTable } from "$lib/utils";

const table = renderTable(element, {
  data: transactions,
  columns: [
    { title: "Date", field: "date" },
    { title: "Account", field: "account" },
    { title: "Amount", field: "amount", formatter: "money" }
  ]
});
```

### Custom Formatters

```javascript
import { formatCurrency, formatDate } from "$lib/utils";

columns: [
  {
    title: "Amount",
    field: "amount",
    formatter: (cell) => formatCurrency(cell.getValue())
  }
]
```

## Charts with D3.js

### Basic Chart Setup

```javascript
import * as d3 from "d3";

function renderChart(element, data) {
  const svg = d3.select(element)
    .append("svg")
    .attr("width", width)
    .attr("height", height);
    
  // Chart implementation
}
```

## Sheet System

The sheet system provides a custom formula language:

```javascript
import { parse, evaluate } from "$lib/sheet";

const ast = parse(formula);
const result = evaluate(ast, environment);
```

## Search Query System

Custom query parser for transaction filtering:

```javascript
import { buildAST } from "$lib/search_query_editor";

const query = buildAST("account:Assets amount>1000");
const filtered = transactions.filter(query.filter);
```

## Theme Support

### CSS Variables

```scss
// colors.scss
:root {
  --color-background: #ffffff;
  --color-text: #363636;
  // ...
}

[data-theme="dark"] {
  --color-background: #1a1a1a;
  --color-text: #f5f5f5;
  // ...
}
```

### Theme Switching

```javascript
import { theme } from "$lib/utils";

function toggleTheme() {
  theme.update(t => t === 'light' ? 'dark' : 'light');
}
```

## Common Patterns

### Loading States

```svelte
{#if loading}
  <Spinner />
{:else if error}
  <div class="notification is-danger">{error}</div>
{:else if data.length === 0}
  <ZeroState message="No data found" />
{:else}
  <!-- Display data -->
{/if}
```

### Form Handling

```svelte
<script>
  async function handleSubmit(event) {
    event.preventDefault();
    loading = true;
    
    try {
      const response = await fetch('/api/save', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData)
      });
      
      if (response.ok) {
        // Handle success
      }
    } catch (error) {
      // Handle error
    } finally {
      loading = false;
    }
  }
</script>

<form on:submit={handleSubmit}>
  <!-- Form fields -->
</form>
```

### Modal Dialogs

```svelte
<script>
  import Modal from "$lib/components/Modal.svelte";
  let showModal = false;
</script>

<Modal bind:show={showModal}>
  <div slot="title">Modal Title</div>
  <div slot="body">
    <!-- Modal content -->
  </div>
</Modal>
```

## Development Workflow

### Running Development Server

```bash
npm run dev
```

### Building for Production

```bash
npm run build
```

### Type Checking

```bash
npm run check
```

### Linting

```bash
npm run lint
```

## Adding New Features

### 1. Create New Page

Create file: `/src/routes/(app)/myfeature/+page.svelte`

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  
  let data = [];
  
  onMount(async () => {
    const response = await fetch('/api/myfeature');
    data = await response.json();
  });
</script>

<section class="section">
  <div class="container">
    <h1 class="title">My Feature</h1>
    <!-- Feature content -->
  </div>
</section>
```

### 2. Add Navigation

Update navbar in appropriate section to include link to new feature.

### 3. Create Components

Add reusable components in `/src/lib/components/`.

### 4. Add Utilities

Add shared functions in `/src/lib/` files.

## Performance Tips

1. **Use Svelte's built-in optimizations**
   - Reactive declarations (`$:`)
   - Component lazy loading
   
2. **Optimize large lists**
   - Virtual scrolling for long lists
   - Pagination for tables
   
3. **Code splitting**
   - Dynamic imports for heavy libraries
   - Route-based code splitting (automatic in SvelteKit)

4. **Asset optimization**
   - Compress images
   - Use appropriate formats
   - Lazy load images

## Debugging

### Browser DevTools

- Use Svelte DevTools extension
- Check Network tab for API calls
- Console for JavaScript errors

### SvelteKit Debug Mode

```javascript
import { dev } from '$app/environment';

if (dev) {
  console.log('Debug info:', data);
}
```

## Testing

### Unit Tests

```javascript
import { render } from '@testing-library/svelte';
import Component from './Component.svelte';

test('renders correctly', () => {
  const { getByText } = render(Component, {
    props: { name: 'Test' }
  });
  
  expect(getByText('Test')).toBeInTheDocument();
});
```

## Best Practices

1. **Component Design**
   - Keep components small and focused
   - Use props for configuration
   - Emit events for parent communication

2. **State Management**
   - Use local state when possible
   - Stores for cross-component state
   - Avoid deeply nested state

3. **Performance**
   - Minimize reactive statements
   - Use key blocks for lists
   - Lazy load heavy components

4. **Accessibility**
   - Use semantic HTML
   - Add ARIA labels
   - Ensure keyboard navigation

5. **Code Organization**
   - Group related files
   - Use consistent naming
   - Extract reusable logic