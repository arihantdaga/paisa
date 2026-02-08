# Paisa Backend Developer Guide

## Overview

The Paisa backend is written in Go and provides a REST API for the frontend. It uses the Gin web framework and SQLite for data storage, while maintaining ledger files as the source of truth.

## Project Structure

```
/cmd                    # Command-line interface
/internal               # Internal packages (not importable by external projects)
  /accounting          # Core accounting logic
  /binary             # Embedded ledger binary
  /cache              # Caching utilities
  /config             # Configuration management
  /generator          # Demo data generator
  /ledger             # Ledger file operations
  /model              # Data models and structures
  /prediction         # Transaction prediction using TF-IDF
  /query              # Query processing
  /scraper            # Market data scrapers
  /server             # HTTP server and API handlers
  /service            # Business logic services
  /taxation           # Tax calculation logic
  /utils              # Utility functions
  /xirr               # XIRR calculation
```

## API Endpoints

### Configuration & Setup
- `GET /api/config` - Get configuration and accounts
- `POST /api/config` - Update configuration
- `POST /api/init` - Initialize demo data
- `POST /api/sync` - Sync journal, prices, and portfolios

### Dashboard & Overview
- `GET /api/dashboard` - Dashboard summary data
- `GET /api/networth` - Net worth calculation
- `GET /api/diagnosis` - System diagnostics

### Assets & Investments
- `GET /api/assets/balance` - Asset balances
- `GET /api/investment` - Investment overview
- `GET /api/gain` - Capital gains summary
- `GET /api/gain/:account` - Gains for specific account
- `GET /api/allocation` - Asset allocation
- `GET /api/portfolio_allocation` - Portfolio allocation

### Income & Expenses
- `GET /api/income` - Income analysis
- `GET /api/expense` - Expense analysis
- `GET /api/budget` - Budget tracking
- `GET /api/cash_flow` - Cash flow analysis
- `GET /api/income_statement` - Income statement

### Transactions
- `GET /api/transaction` - All transactions
- `GET /api/transaction/balanced` - Balanced postings
- `GET /api/recurring` - Recurring transactions
- `GET /api/ledger` - Ledger entries

### Liabilities
- `GET /api/liabilities/balance` - Liability balances
- `GET /api/liabilities/interest` - Interest calculations
- `GET /api/liabilities/repayment` - Repayment schedule
- `GET /api/credit_cards` - Credit card summary
- `GET /api/credit_cards/:account` - Specific card details

### Market Data & Pricing
- `GET /api/price` - Price data
- `GET /api/price/providers` - Available price providers
- `POST /api/price/delete` - Clear price cache
- `POST /api/price/providers/delete/:provider` - Clear specific provider cache
- `POST /api/price/autocomplete` - Price code autocomplete

### Tax & Reports
- `GET /api/capital_gains` - Capital gains report
- `GET /api/harvest` - Tax loss harvesting opportunities
- `GET /api/schedule_al` - Schedule AL for tax filing

### File Management
- `GET /api/editor/files` - List ledger files
- `POST /api/editor/file` - Get file content
- `POST /api/editor/validate` - Validate ledger file
- `POST /api/editor/save` - Save file changes
- `POST /api/editor/file/delete_backups` - Delete file backups

### Sheets
- `GET /api/sheets/files` - List sheet files
- `POST /api/sheets/file` - Get sheet content
- `POST /api/sheets/save` - Save sheet
- `POST /api/sheets/file/delete_backups` - Delete sheet backups

### Other
- `GET /api/logs` - Application logs
- `GET /api/account/tf_idf` - TF-IDF for account prediction
- `GET /api/templates` - Import templates
- `POST /api/templates/upsert` - Create/update template
- `POST /api/templates/delete` - Delete template
- `GET /api/goals` - Financial goals
- `GET /api/goals/:type/:name` - Specific goal details

## Key Components

### 1. Server Setup (`internal/server/server.go`)

```go
func Build(db *gorm.DB, enableCompression bool) *gin.Engine {
    router := gin.New()
    
    // Middleware
    router.Use(Logger(log.StandardLogger()), gin.Recovery())
    router.Use(TokenAuthMiddleware())
    
    // API routes
    router.GET("/api/ping", pingHandler)
    // ... more routes
    
    return router
}
```

### 2. Database Connection (`internal/utils/utils.go`)

```go
func OpenDB() (*gorm.DB, error) {
    db, err := gorm.Open(sqlite.Open(config.GetDBPath()), &gorm.Config{
        Logger: gorm_logrus.New(),
    })
    return db, err
}
```

### 3. Configuration (`internal/config/config.go`)

The configuration system supports:
- YAML configuration files
- Environment variable overrides
- JSON schema validation
- Hot reloading

### 4. Ledger Integration (`internal/ledger/ledger.go`)

- Executes ledger-cli commands
- Parses ledger output
- Validates transactions
- Handles multiple ledger file formats (ledger, hledger, beancount)

### 5. Data Models (`internal/model/`)

Key models include:
- `Transaction`: Financial transactions
- `Posting`: Individual posting within a transaction
- `Price`: Market prices for commodities
- `Commodity`: Stocks, mutual funds, currencies
- `Portfolio`: Investment portfolio entries

## Creating New API Endpoints

1. **Define the handler** in appropriate file under `/internal/server/`:

```go
func GetMyData(db *gorm.DB) gin.H {
    // Business logic here
    var data []MyModel
    db.Find(&data)
    
    return gin.H{"data": data}
}
```

2. **Add route** in `server.go`:

```go
router.GET("/api/mydata", func(c *gin.Context) {
    c.JSON(200, GetMyData(db))
})
```

3. **Add request/response types** if needed:

```go
type MyDataRequest struct {
    Filter string `json:"filter"`
}

type MyDataResponse struct {
    Data    []MyModel `json:"data"`
    Success bool      `json:"success"`
}
```

## Working with Ledger Files

### Reading Transactions

```go
transactions, err := ledger.ParseFile(filepath)
```

### Executing Ledger Commands

```go
result, err := ledger.Exec([]string{"balance", "Assets"})
```

### File Watching

The system watches ledger files for changes and automatically syncs data.

## Database Operations

### Using GORM

```go
// Find all transactions
var transactions []model.Transaction
db.Find(&transactions)

// Query with conditions
db.Where("date > ?", startDate).Find(&transactions)

// Preload associations
db.Preload("Postings").Find(&transactions)
```

### Caching Strategy

- SQLite stores parsed ledger data
- Prices are cached with expiration
- Cache is invalidated on file changes

## Adding Price Providers

1. **Implement the provider interface**:

```go
type PriceProvider interface {
    GetPrice(code string) (decimal.Decimal, error)
    Name() string
}
```

2. **Register in** `internal/scraper/`:

```go
func RegisterProvider(provider PriceProvider) {
    providers[provider.Name()] = provider
}
```

## Error Handling

- Use structured logging with logrus
- Return appropriate HTTP status codes
- Include error details in responses:

```go
c.JSON(http.StatusBadRequest, gin.H{
    "error": err.Error(),
    "success": false,
})
```

## Testing

Run tests with:

```bash
go test ./...
```

Key test files:
- `internal/ledger/ledger_test.go`
- `internal/utils/utils_test.go`
- `internal/xirr/xirr_test.go`

## Performance Considerations

1. **Batch Operations**: Process multiple items together
2. **Caching**: Use SQLite for frequently accessed data
3. **Lazy Loading**: Load data only when needed
4. **Indexing**: Ensure proper database indexes

## Security

- Token-based authentication
- Input validation on all endpoints
- File path sanitization
- Read-only mode support
- CORS configuration

## Common Tasks

### Adding a New Report

1. Create handler in `/internal/server/`
2. Add data processing logic
3. Define response structure
4. Add route in `server.go`
5. Update frontend to consume API

### Debugging

- Enable debug logging: `LOG_LEVEL=debug`
- Check `/api/logs` endpoint
- Use `/api/diagnosis` for system health

### Database Migrations

Currently manual - modify schema in model definitions and recreate database.

## Best Practices

1. **Keep handlers thin** - Move logic to service layer
2. **Use transactions** for data consistency
3. **Validate input** before processing
4. **Log important operations**
5. **Handle errors gracefully**
6. **Write tests** for new functionality
7. **Document API changes**