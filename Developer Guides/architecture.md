# Paisa Architecture Overview

## Introduction

Paisa is a personal finance manager built on top of the ledger double-entry accounting tool. It provides a modern web interface for managing personal finances with features like expense tracking, investment management, budgeting, and more.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        User Interface                         │
│                    (SvelteKit Web App)                       │
├─────────────────────────────────────────────────────────────┤
│                      REST API Layer                          │
│                    (Gin HTTP Server)                         │
├─────────────────────────────────────────────────────────────┤
│                    Business Logic Layer                       │
│                  (Go Internal Packages)                      │
├─────────────────────────────────────────────────────────────┤
│                      Data Layer                              │
│              ┌────────────┐  ┌────────────┐                 │
│              │   SQLite   │  │   Ledger   │                 │
│              │  Database  │  │   Files    │                 │
│              └────────────┘  └────────────┘                 │
└─────────────────────────────────────────────────────────────┘
```

## Technology Stack

### Backend
- **Language**: Go (Golang)
- **Web Framework**: Gin
- **Database**: SQLite (via GORM ORM)
- **Ledger**: Uses ledger-cli binary for transaction processing
- **Authentication**: Token-based authentication

### Frontend
- **Framework**: SvelteKit (Svelte 4)
- **Build Tool**: Vite
- **Styling**: Bulma CSS framework + Tailwind CSS + Custom SCSS
- **UI Components**: 
  - Tabulator for data tables
  - D3.js for visualizations
  - CodeMirror for code editing
- **State Management**: Svelte stores with localStorage persistence

### Desktop Application
- **Framework**: Wails v2 (Go + Web Technologies)
- Embeds the web interface in a native desktop application

## Core Components

### 1. Command-Line Interface (`/cmd`)
- **root.go**: Main CLI entry point using Cobra
- **serve.go**: Starts the web server
- **init.go**: Initializes a new ledger journal
- **update.go**: Updates prices and market data
- **version.go**: Version information

### 2. Internal Packages (`/internal`)

#### Accounting (`/internal/accounting`)
- Core accounting logic and account tree management
- Profit & Loss calculations
- Account behaviors and classifications

#### Server (`/internal/server`)
- HTTP server setup and middleware
- API endpoint handlers
- Request/response handling

#### Model (`/internal/model`)
- Data models for transactions, postings, prices, commodities
- Import templates for various financial institutions
- Portfolio and mutual fund scheme data structures

#### Ledger (`/internal/ledger`)
- Interface with ledger-cli binary
- Parsing ledger file format
- Transaction validation

#### Scraper (`/internal/scraper`)
- Market data scrapers for:
  - Mutual funds (India)
  - Stocks (Yahoo Finance, Alpha Vantage)
  - NPS (National Pension Scheme)
  - Metals
  - CII (Cost Inflation Index)

### 3. Web Interface (`/src`)
- SvelteKit application with file-based routing
- Modular component architecture
- Custom parsers for sheets and search queries

## Key Features Architecture

### Transaction Management
- Ledger files are the source of truth
- SQLite caches parsed data for performance
- Real-time validation using ledger-cli

### Market Data & Pricing
- Pluggable price providers
- Automatic price updates via scrapers
- Historical price caching in SQLite

### Import System
- Template-based import using Handlebars
- Support for multiple file formats (CSV, XLS, PDF)
- Extensible for new financial institutions

### Sheet System
- Custom formula language for financial calculations
- Real-time evaluation in the browser
- Integration with transaction data

### Search & Query
- Custom query language with parser
- Advanced filtering capabilities
- Integration with all data types

## Data Flow

1. **User Input** → Web UI → API Request
2. **API Handler** → Business Logic → Data Access
3. **Data Access** → Read from Ledger files or SQLite cache
4. **Processing** → Apply business rules, calculations
5. **Response** → Format data → Send to UI
6. **UI Update** → Render visualizations and tables

## Security Considerations

- Token-based authentication for API access
- Read-only mode support
- File system access restrictions
- Input validation at all layers

## Deployment Options

1. **Local Server**: Run on personal machine
2. **Docker**: Containerized deployment
3. **Desktop App**: Native application via Wails
4. **Web Hosting**: Deploy as web service

## Extension Points

- Custom import templates
- New price providers
- Additional report types
- Custom sheet functions
- Theme customization