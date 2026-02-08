# Paisa Parsers Guide

## Overview

Paisa includes several parsers for different purposes:
1. **Import Parsers** - Convert bank statements to ledger format
2. **Sheet Parser** - Custom formula language for calculations
3. **Search Query Parser** - Advanced transaction filtering
4. **Ledger Parser** - Parse ledger file formats

## Import Parsers

### Architecture

Import parsers use Handlebars templates to transform various file formats (CSV, XLS, PDF) into ledger transactions.

### Template Location

- Built-in: `/internal/model/template/templates/`
- Custom: `~/.paisa/templates/`

### Template Structure

```handlebars
{{#each table}}
{{#if (isPositive "Debit")}}
{{yyyy-mm-dd Date}} {{Payee}}
    {{Account}}  {{currency Debit}}
    Assets:Checking:HDFC
{{/if}}
{{/each}}
```

### Available Helpers

#### Date Formatting
- `{{yyyy-mm-dd date}}` - Format as YYYY-MM-DD
- `{{dd-mm-yyyy date}}` - Format as DD-MM-YYYY
- `{{parseDate "format" dateStr}}` - Parse custom date format

#### Number Handling
- `{{currency amount}}` - Format with currency symbol
- `{{abs number}}` - Absolute value
- `{{isPositive number}}` - Check if positive
- `{{round number decimals}}` - Round to decimals

#### String Operations
- `{{trim string}}` - Remove whitespace
- `{{replace "find" "replace" string}}` - Replace text
- `{{uppercase string}}` - Convert to uppercase
- `{{lowercase string}}` - Convert to lowercase

#### Logic
- `{{#if condition}}...{{/if}}`
- `{{#unless condition}}...{{/unless}}`
- `{{#each array}}...{{/each}}`
- `{{#equal a b}}...{{/equal}}`

### Creating Custom Import Templates

1. **Analyze the source format**
```csv
Date,Description,Debit,Credit,Balance
01-15-2024,Grocery Store,50.00,,1000.00
01-16-2024,Salary,,3000.00,4000.00
```

2. **Create template**
```handlebars
{{#each table}}
{{#if (isPositive Debit)}}
{{yyyy-mm-dd (parseDate "MM-DD-YYYY" Date)}} {{Description}}
    Expenses:Groceries  {{currency Debit}}
    Assets:Checking
{{else if (isPositive Credit)}}
{{yyyy-mm-dd (parseDate "MM-DD-YYYY" Date)}} {{Description}}
    Assets:Checking  {{currency Credit}}
    Income:Salary
{{/if}}
{{/each}}
```

3. **Save and use**
```bash
# Save to ~/.paisa/templates/mybank.yaml
name: "My Bank Statement"
content: |
  {{template content here}}
```

### Supported File Formats

#### CSV Files
- Direct parsing with column headers
- Custom delimiters supported

#### Excel Files (XLS/XLSX)
- Multiple sheet support
- Named ranges
- Cell references

#### PDF Files
- Table extraction
- Text parsing
- OCR support (if configured)

## Sheet Parser

### Overview

The sheet parser provides a custom formula language for financial calculations, similar to spreadsheet formulas.

### Grammar Structure

Located in `/src/lib/sheet/language.grammar`

```
@top Sheet { Line* }

Line { Expression? LineEnd }

Expression {
  Number |
  String |
  Boolean |
  FunctionCall |
  BinaryExpression |
  UnaryExpression |
  ParenthesizedExpression |
  Reference
}
```

### Supported Functions

#### Aggregation Functions
- `SUM(query)` - Sum matching transactions
- `COUNT(query)` - Count transactions
- `AVG(query)` - Average amount
- `MIN(query)` - Minimum value
- `MAX(query)` - Maximum value

#### Date Functions
- `TODAY()` - Current date
- `MONTH(date)` - Extract month
- `YEAR(date)` - Extract year
- `DAYS(date1, date2)` - Days between dates

#### Financial Functions
- `XIRR(query)` - Extended Internal Rate of Return
- `BALANCE(account)` - Account balance
- `BUDGET(category)` - Budget amount

#### Query Syntax
```
SUM(account:"Expenses:Food" date:"this month")
COUNT(account:"Assets" amount:">1000")
BALANCE("Assets:Investment")
```

### Parser Implementation

Located in `/src/lib/sheet/interpreter.ts`

```typescript
export class Environment {
  scope: Record<string, any>;
  postings: Posting[];
  
  evaluate(ast: AST): any {
    switch (ast.type) {
      case "FunctionCall":
        return this.evaluateFunction(ast);
      case "BinaryExpression":
        return this.evaluateBinary(ast);
      // ...
    }
  }
}
```

### Adding New Functions

1. **Update grammar** (`language.grammar`):
```
FunctionName {
  "SUM" | "COUNT" | "AVG" | "MYNEWFUNC"
}
```

2. **Add to functions.ts**:
```typescript
export const MYNEWFUNC: SheetFunction = {
  name: "MYNEWFUNC",
  arity: [1, 2], // Min and max arguments
  evaluate: (env: Environment, args: any[]) => {
    // Implementation
    return result;
  }
};
```

3. **Register function**:
```typescript
const functions = {
  SUM,
  COUNT,
  MYNEWFUNC,
  // ...
};
```

## Search Query Parser

### Overview

Provides advanced filtering for transactions using a custom query language.

### Grammar

Located in `/src/lib/search/parser/query.grammar`

```
@top Query { Expression }

Expression {
  AndExpression |
  OrExpression |
  NotExpression |
  Condition
}

Condition {
  AccountCondition |
  AmountCondition |
  DateCondition |
  PayeeCondition
}
```

### Query Examples

```
# Simple queries
account:Assets
amount:>1000
date:"last month"
payee:"Amazon"

# Combined queries
account:Expenses AND amount:>50
(account:Assets OR account:Liabilities) AND date:2024

# Negation
NOT account:Expenses:Entertainment
```

### Parser Implementation

Located in `/src/lib/search_query_editor.ts`

```typescript
export function buildAST(query: string): QueryAST {
  const tree = parser.parse(query);
  return processNode(tree.topNode);
}

export class QueryAST {
  filter(transaction: Transaction): boolean {
    // Apply filters
  }
}
```

### Supported Operators

#### Comparison
- `=` - Equals
- `>` - Greater than
- `<` - Less than
- `>=` - Greater or equal
- `<=` - Less or equal
- `!=` - Not equal

#### Logical
- `AND` - Both conditions
- `OR` - Either condition
- `NOT` - Negation

#### Special
- `:` - Contains/matches
- `~` - Regex match

## Ledger File Parser

### Overview

Parses ledger, hledger, and beancount file formats.

### Implementation

Located in `/internal/ledger/ledger.go`

```go
func ParseFile(path string) ([]model.Posting, error) {
    // Detect format
    format := detectFormat(path)
    
    // Execute appropriate parser
    switch format {
    case "ledger":
        return parseLedger(path)
    case "beancount":
        return parseBeancount(path)
    case "hledger":
        return parseHledger(path)
    }
}
```

### Format Detection

```go
func detectFormat(path string) string {
    ext := filepath.Ext(path)
    switch ext {
    case ".beancount":
        return "beancount"
    case ".hledger":
        return "hledger"
    default:
        return "ledger"
    }
}
```

### Transaction Structure

```
2024-01-15 * Payee Description
    ; Comment
    Account:Name    $100.00
    Other:Account  -$100.00
    ; :tag:value:
```

## Parser Development Tips

### 1. Using Lezer for Grammar

```bash
# Generate parser from grammar
npm run parser-build

# Debug mode with names
npm run parser-build-debug
```

### 2. Testing Parsers

```typescript
// Test sheet parser
import { parse } from "$lib/sheet";

test("parses function call", () => {
  const ast = parse("SUM(account:Assets)");
  expect(ast.type).toBe("FunctionCall");
});
```

### 3. Error Handling

```typescript
try {
  const result = parse(input);
} catch (error) {
  if (error instanceof ParseError) {
    console.log(`Error at position ${error.position}`);
  }
}
```

### 4. Performance Optimization

- Cache parsed results
- Use incremental parsing for editors
- Minimize AST traversals

## Debugging Parsers

### Sheet Parser Debugging

```typescript
// Enable debug logging
const DEBUG = true;

function evaluate(ast: AST, env: Environment) {
  if (DEBUG) {
    console.log("Evaluating:", ast);
  }
  // ...
}
```

### Import Parser Testing

```bash
# Test template with sample file
paisa import --template mytemplate.yaml sample.csv --dry-run
```

### Query Parser Visualization

```typescript
// Visualize AST
function printAST(node: SyntaxNode, indent = 0) {
  console.log(" ".repeat(indent) + node.type.name);
  for (let child = node.firstChild; child; child = child.nextSibling) {
    printAST(child, indent + 2);
  }
}
```

## Common Parser Patterns

### 1. Recursive Descent

```typescript
function parseExpression(): AST {
  const left = parseTerm();
  
  while (match("+", "-")) {
    const op = previous();
    const right = parseTerm();
    left = new BinaryOp(left, op, right);
  }
  
  return left;
}
```

### 2. Visitor Pattern

```typescript
abstract class Visitor<T> {
  abstract visitNumber(node: NumberAST): T;
  abstract visitBinary(node: BinaryAST): T;
  abstract visitFunction(node: FunctionAST): T;
}
```

### 3. Error Recovery

```typescript
function synchronize() {
  advance();
  
  while (!isAtEnd()) {
    if (previous().type === SEMICOLON) return;
    
    switch (peek().type) {
      case IF:
      case FOR:
      case RETURN:
        return;
    }
    
    advance();
  }
}
```

## Best Practices

1. **Grammar Design**
   - Keep grammar simple and unambiguous
   - Use precedence rules for operators
   - Provide good error messages

2. **Parser Implementation**
   - Separate lexing from parsing
   - Use immutable AST nodes
   - Cache results when possible

3. **Testing**
   - Test edge cases
   - Fuzz test with random input
   - Benchmark performance

4. **Documentation**
   - Document grammar rules
   - Provide usage examples
   - Include error scenarios