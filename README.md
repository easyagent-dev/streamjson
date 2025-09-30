# LLMJson - Streaming JSON Parser for Go

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.24.4+-00ADD8?style=flat&logo=go)](https://golang.org/)

A high-performance, memory-efficient streaming JSON parser designed for processing JSON data incrementally. Perfect for handling JSON responses from APIs, especially Large Language Models (LLMs), where data arrives in chunks and you want to access values as soon as they become available.

## Features

- **🚀 Streaming Processing**: Parse JSON data incrementally as it arrives
- **💾 Memory Efficient**: Object pooling and optimized memory management
- **🛡️ Error Tolerant**: Continues parsing even when encountering invalid tokens
- **🔍 Partial Access**: Access values before complete JSON is received
- **🌳 AST Building**: Builds Abstract Syntax Tree for efficient data access
- **⚡ High Performance**: Optimized tokenization and parsing algorithms
- **🎯 Type Safe**: Proper Go type conversion for JSON values

## Installation

```bash
go get github.com/easymvp/llmjson
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/easymvp/llmjson"
)

func main() {
    parser := llmjson.NewStreamJSONParser()
    
    // Parse complete JSON
    parser.Append(`{"name":"John","age":30,"active":true}`)
    
    // Access values
    name := parser.Get("name")        // "John"
    age := parser.Get("age")          // int64(30)
    active := parser.Get("active")    // true
    
    fmt.Printf("Name: %s, Age: %d, Active: %t\n", name, age, active)
}
```

## Usage Examples

### Incremental Parsing

Perfect for streaming scenarios where JSON arrives in chunks:

```go
parser := llmjson.NewStreamJSONParser()

// JSON arrives in multiple chunks
parser.Append(`{"user":{`)
parser.Append(`"name":"Alice"`)
parser.Append(`,"age":25`)
parser.Append(`}}`)

// Access nested values
name := parser.Get("user", "name")  // "Alice"
age := parser.Get("user", "age")    // int64(25)

// Check if parsing is complete
if parser.IsCompleted() {
    fmt.Println("JSON parsing completed")
}
```

### Partial Value Access

Access values as soon as they become available:

```go
parser := llmjson.NewStreamJSONParser()

parser.Append(`{"message":"Hello`)
// Partial string content is available
msg := parser.Get("message")  // "Hello"

parser.Append(` World","status":"`)
// Now message is complete
msg = parser.Get("message")   // "Hello World"

parser.Append(`success"}`)
status := parser.Get("status") // "success"
```

### Array Processing

Handle arrays with indexed access:

```go
parser := llmjson.NewStreamJSONParser()

parser.Append(`[{"id":1,"name":"Item1"},{"id":2,"name":"Item2"}]`)

// Access array elements by index
firstItem := parser.Get("0", "name")   // "Item1"
secondId := parser.Get("1", "id")      // int64(2)
```

### Error Tolerance

Parser continues working even with invalid data:

```go
parser := llmjson.NewStreamJSONParser()

// Invalid tokens are tolerated
parser.Append(`invalid text {"valid": true}`)

valid := parser.Get("valid")  // true
```

### Complex Nested Structures

```go
parser := llmjson.NewStreamJSONParser()

jsonData := `{
    "response": {
        "users": [
            {"id": 1, "profile": {"name": "Alice", "settings": {"theme": "dark"}}},
            {"id": 2, "profile": {"name": "Bob", "settings": {"theme": "light"}}}
        ],
        "meta": {
            "total": 2,
            "page": 1
        }
    }
}`

parser.Append(jsonData)

// Deep nested access
aliceName := parser.Get("response", "users", "0", "profile", "name")           // "Alice"
bobTheme := parser.Get("response", "users", "1", "profile", "settings", "theme") // "light"
total := parser.Get("response", "meta", "total")                               // int64(2)
```

## API Reference

### StreamJSONParser

#### Constructor

```go
func NewStreamJSONParser() *StreamJSONParser
```
Creates a new streaming JSON parser instance.

#### Methods

```go
func (p *StreamJSONParser) Append(content string)
```
Appends content to the parser buffer and processes available tokens.

```go
func (p *StreamJSONParser) Get(keys ...string) interface{}
```
Retrieves a value from the parsed JSON using a path of keys. Returns `nil` if the path doesn't exist or the value isn't available yet.

```go
func (p *StreamJSONParser) IsCompleted() bool
```
Returns `true` if all JSON structures have been properly closed and parsing is complete.

```go
func (p *StreamJSONParser) GetRoot() *Node
```
Returns the root node of the Abstract Syntax Tree.

### Node Types

The parser builds an AST with three node types:

- **ObjectNode**: Represents JSON objects `{}`
- **ArrayNode**: Represents JSON arrays `[]`
- **ValueNode**: Represents primitive values (string, number, boolean, null)

### Value Types

The parser converts JSON values to appropriate Go types:

- **Strings**: `string`
- **Numbers**: `int64` (integers) or `float64` (floating-point)
- **Booleans**: `bool`
- **Null**: `nil`
- **Objects/Arrays**: `*Node`

## Performance Considerations

- Uses object pooling to minimize garbage collection
- Efficient byte-level processing for tokenization
- Pre-allocated buffers for optimal memory usage
- Minimal string allocations during parsing
- Fast character-by-character processing with optimized lookups

## Use Cases

- **LLM Response Processing**: Parse JSON responses from language models as they stream
- **API Streaming**: Handle chunked JSON responses from REST APIs
- **Real-time Data**: Process JSON data in real-time applications
- **Large JSON Files**: Parse large JSON files without loading everything into memory
- **WebSocket Messages**: Process JSON messages from WebSocket connections
- **Log Processing**: Parse JSON log entries as they're written

## Memory Management

The parser includes automatic memory management:

```go
// Nodes are automatically returned to object pools
// No manual cleanup required
parser := llmjson.NewStreamJSONParser()
// ... use parser
// Memory is automatically reclaimed
```

## Error Handling

The parser is designed to be fault-tolerant:

- Invalid tokens are skipped rather than causing errors
- Partial JSON can be processed
- Malformed input doesn't crash the parser
- Incomplete values return `nil` until they're complete

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests.

---

**Note**: This library is optimized for streaming scenarios where JSON data arrives incrementally. For simple, complete JSON parsing, the standard library's `encoding/json` package might be more appropriate.
