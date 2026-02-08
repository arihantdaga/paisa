# Paisa Data Storage Guide

## Overview

Paisa uses a hybrid storage approach:
- **Ledger files** (.ledger, .beancount, .hledger) as the primary source of truth
- **SQLite database** for caching, performance optimization, and additional metadata
- **YAML/JSON files** for configuration and templates

## Storage Architecture

```
┌─────────────────────────────────┐
│      User's Ledger Files        │  ← Source of Truth
│   (Plain text accounting)       │
└────────────┬────────────────────┘
             │ Parse & Import
             ↓
┌─────────────────────────────────┐
│       SQLite Database           │  ← Cache & Metadata
│   (Parsed data, prices, etc)    │
└─────────────────────────────────┘
```

## File Storage

### 1. Ledger Files

**Location**: User-defined (configured in `paisa.yaml`)

**Supported Formats**:
- Ledger CLI format (`.ledger`, `.journal`)
- Beancount format (`.beancount`)
- hledger format (`.hledger`)

**Example Structure**:
```
/path/to/journals/
├── main.ledger          # Main journal file
├── accounts.ledger      # Account definitions
├── prices.ledger        # Price data
└── transactions/        # Transaction files
    ├── 2024-01.ledger
    ├── 2024-02.ledger
    └── ...
```

### 2. Configuration Files

**Location**: `~/.paisa/` (or custom path via `PAISA_CONFIG` env var)

```
~/.paisa/
├── paisa.yaml           # Main configuration
├── paisa.db            # SQLite database
├── templates/          # Import templates
│   ├── custom1.yaml
│   └── custom2.yaml
└── sheets/             # Sheet files
    ├── budget.paisa
    └── analysis.paisa
```

### 3. Sheet Files

**Location**: `~/.paisa/sheets/`

Custom spreadsheet-like files with `.paisa` extension containing formulas and calculations.

## SQLite Database Schema

### Core Tables

#### 1. `postings`
Stores parsed posting data from ledger files.

```sql
CREATE TABLE postings (
    id INTEGER PRIMARY KEY,
    transaction_id TEXT,
    date DATE,
    payee TEXT,
    account TEXT,
    commodity TEXT,
    quantity DECIMAL,
    amount DECIMAL,
    market_amount DECIMAL,
    status TEXT,
    tag_recurring TEXT,
    tag_period TEXT,
    transaction_begin_line INTEGER,
    transaction_end_line INTEGER,
    file_name TEXT,
    forecast BOOLEAN,
    note TEXT,
    transaction_note TEXT,
    UNIQUE(file_name, transaction_begin_line, account)
);
```

#### 2. `prices`
Market price data for commodities.

```sql
CREATE TABLE prices (
    id INTEGER PRIMARY KEY,
    date DATE,
    commodity_type TEXT,
    commodity_id TEXT,
    commodity_name TEXT,
    value DECIMAL,
    UNIQUE(date, commodity_type, commodity_id)
);
```

#### 3. `portfolios`
Portfolio holdings and transactions.

```sql
CREATE TABLE portfolios (
    id INTEGER PRIMARY KEY,
    account TEXT,
    commodity_type TEXT,
    commodity_id TEXT,
    commodity_name TEXT,
    quantity DECIMAL,
    purchase_date DATE,
    purchase_price DECIMAL,
    market_value DECIMAL,
    market_value_date DATE,
    xirr DECIMAL
);
```

#### 4. `networth_breakdowns`
Historical networth calculations.

```sql
CREATE TABLE networth_breakdowns (
    id INTEGER PRIMARY KEY,
    account TEXT,
    date DATE,
    amount DECIMAL,
    UNIQUE(account, date)
);
```

#### 5. `templates`
Import templates for various file formats.

```sql
CREATE TABLE templates (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE,
    content TEXT
);
```

### Cache Tables

#### `mutual_fund_schemes`
```sql
CREATE TABLE mutual_fund_schemes (
    id INTEGER PRIMARY KEY,
    code TEXT UNIQUE,
    name TEXT,
    isin TEXT,
    type TEXT,
    category TEXT,
    amc TEXT
);
```

#### `nps_schemes`
```sql
CREATE TABLE nps_schemes (
    id INTEGER PRIMARY KEY,
    pfm_id TEXT,
    pfm_name TEXT,
    scheme_id TEXT,
    scheme_name TEXT,
    UNIQUE(pfm_id, scheme_id)
);
```

#### `cost_inflation_index`
```sql
CREATE TABLE cost_inflation_index (
    id INTEGER PRIMARY KEY,
    year INTEGER UNIQUE,
    value INTEGER
);
```

## Data Flow

### 1. Initial Load

```
1. Read paisa.yaml configuration
2. Parse all ledger files specified
3. Store parsed data in SQLite
4. Fetch market prices if needed
5. Calculate derived data (networth, gains, etc)
```

### 2. Sync Process

```go
type SyncRequest struct {
    Journal    bool // Re-parse ledger files
    Prices     bool // Update market prices
    Portfolios bool // Recalculate portfolios
}
```

### 3. File Watching

The system watches ledger files for changes and automatically triggers sync.

## Data Models

### Transaction Structure

```go
type Transaction struct {
    ID             string
    Date           time.Time
    Payee          string
    Postings       []Posting
    TagRecurring   string
    TagPeriod      string
    BeginLine      int
    EndLine        int
    FileName       string
    Forecast       bool
    Note           string
}
```

### Posting Structure

```go
type Posting struct {
    ID              int
    TransactionID   string
    Date            time.Time
    Payee           string
    Account         string
    Commodity       string
    Quantity        decimal.Decimal
    Amount          decimal.Decimal
    MarketAmount    decimal.Decimal
    Status          string
    // ... other fields
}
```

## Working with Data

### Reading from Ledger Files

```go
// Direct ledger command execution
output, err := ledger.Exec([]string{"balance", "Assets"})

// Parse ledger file
postings, err := ledger.ParseFile("main.ledger")
```

### Querying SQLite

```go
// Using GORM
var postings []model.Posting
db.Where("account LIKE ?", "Assets:%").Find(&postings)

// Raw SQL
db.Raw(`
    SELECT account, SUM(amount) as total 
    FROM postings 
    WHERE date >= ? 
    GROUP BY account
`, startDate).Scan(&results)
```

### Caching Strategy

1. **Posting Data**: Cached indefinitely, invalidated on file change
2. **Price Data**: Cached with TTL, refreshed periodically
3. **Calculations**: Cached per request, cleared on data change

## File Format Examples

### Ledger Format

```ledger
2024-01-15 * Grocery Store
    Expenses:Food:Groceries         $50.00
    Assets:Checking                -$50.00

2024-01-15 * Investment
    Assets:Investment:MutualFund    10 FUND123 @ $100.00
    Assets:Checking              -$1000.00
```

### Configuration (paisa.yaml)

```yaml
journal_path: /home/user/documents/main.ledger
db_path: /home/user/.paisa/paisa.db
readonly: false
default_currency: USD
financial_year_starting_month: 4

commodities:
  - name: FUND123
    type: mutualfund
    price:
      provider: in-mfapi
      code: "123456"

accounts:
  - name: "Assets:Checking"
    icon: "mdi:bank"
```

### Sheet Format

```
# Budget Analysis
income = SUM(account:"Income:Salary" date:"this month")
expenses = SUM(account:"Expenses" date:"this month")
savings = income - expenses
savings_rate = (savings / income) * 100
```

## Backup and Recovery

### Backup Strategy

1. **Ledger Files**: Version control (Git) recommended
2. **Database**: Regular SQLite backups
3. **Configuration**: Include in version control

### Recovery Process

1. Restore ledger files from backup
2. Delete SQLite database
3. Run sync to rebuild cache

## Performance Optimization

### Indexes

Key indexes for performance:

```sql
CREATE INDEX idx_postings_date ON postings(date);
CREATE INDEX idx_postings_account ON postings(account);
CREATE INDEX idx_postings_commodity ON postings(commodity);
CREATE INDEX idx_prices_date_commodity ON prices(date, commodity_id);
```

### Query Optimization

1. **Use account prefixes** for efficient filtering
2. **Date ranges** to limit data scope
3. **Aggregations** in database rather than application

## Data Integrity

### Validation

1. **Ledger validation**: Uses ledger-cli for validation
2. **Database constraints**: UNIQUE constraints prevent duplicates
3. **Transaction atomicity**: All operations wrapped in transactions

### Consistency Checks

- Balance assertions in ledger files
- Database foreign key constraints
- Periodic reconciliation checks

## Migration and Updates

### Schema Updates

Currently manual process:
1. Backup existing database
2. Update model definitions
3. Recreate database with new schema
4. Re-sync data

### Data Migration

For migrating between ledger formats:
```bash
# Convert ledger to beancount
ledger2beancount main.ledger > main.beancount

# Convert hledger to ledger
hledger -f main.hledger print > main.ledger
```

## Best Practices

1. **Keep ledger files organized** with clear naming
2. **Use includes** for modular ledger files
3. **Regular backups** of both files and database
4. **Version control** for ledger files
5. **Validate before syncing** to catch errors early
6. **Monitor database size** and vacuum periodically
7. **Use transactions** for data consistency

## Troubleshooting

### Common Issues

1. **Sync failures**: Check ledger file syntax
2. **Missing data**: Verify file paths in config
3. **Performance issues**: Check database indexes
4. **Duplicate entries**: Review UNIQUE constraints

### Debug Commands

```bash
# Check database integrity
sqlite3 paisa.db "PRAGMA integrity_check;"

# View table structure
sqlite3 paisa.db ".schema postings"

# Export data
sqlite3 paisa.db ".dump" > backup.sql
```