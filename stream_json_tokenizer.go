// Copyright 2025 easymvp
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package llmjson

import (
	"strings"
)

// TokenType represents the type of JSON token
type TokenType int

const (
	ObjectStart TokenType = iota // {
	ObjectEnd                    // }
	ArrayStart                   // [
	ArrayEnd                     // ]
	Number                       // 123, 45.67, -8.9e-10
	Bool                         // true, false
	ObjectKey                    // "key" (string used as object key)
	String                       // "value" (string value)
	Colon                        // :
	Comma                        // ,
	Null                         // null
	EOF                          // End of input
	Invalid                      // Invalid token
)

// Token represents a JSON token
type Token struct {
	TokenStart int       // Start position in the input
	TokenEnd   int       // End position in the input
	TokenType  TokenType // Type of the token
	Content    string    // Content of the token
	Completed  bool      // Whether the token is complete
}

// StreamJSONTokenizer implements a streaming JSON tokenizer
type StreamJSONTokenizer struct {
	buffer       []byte // Input buffer using bytes for efficiency
	position     int    // Current position in buffer
	lastToken    *Token // Last incomplete token
	escapeNext   bool   // Whether next character is escaped
	expectingKey bool   // Whether we're expecting an object key

	// Pre-allocated string builder for efficient string construction
	contentBuilder strings.Builder
}

// Predefined constants to avoid allocations
var (
	singleChars = [256]string{} // Pre-allocated single character strings
	trueBytes   = []byte("true")
	falseBytes  = []byte("false")
	nullBytes   = []byte("null")
)

// Initialize single character strings once
func init() {
	for i := 0; i < 256; i++ {
		singleChars[i] = string(byte(i))
	}
}

// NewStreamJSONTokenizer creates a new streaming JSON tokenizer
func NewStreamJSONTokenizer() *StreamJSONTokenizer {
	tokenizer := &StreamJSONTokenizer{
		buffer:       make([]byte, 0, 1024), // Pre-allocate with reasonable capacity
		position:     0,
		expectingKey: false,
	}
	tokenizer.contentBuilder.Grow(256) // Pre-allocate builder capacity
	return tokenizer
}

// Append adds more content to the tokenizer
func (t *StreamJSONTokenizer) Append(content string) {
	// Use append instead of string concatenation for better performance
	t.buffer = append(t.buffer, content...)
}

// NextToken returns the next token from the input
func (t *StreamJSONTokenizer) NextToken() Token {
	// If we have an incomplete token, try to complete it
	if t.lastToken != nil && !t.lastToken.Completed {
		token := t.continueToken()
		if token.Completed {
			t.lastToken = nil
		} else {
			t.lastToken = &token
		}
		return token
	}

	// Skip whitespace
	t.skipWhitespace()

	// Check if we've reached the end
	if t.position >= len(t.buffer) {
		return Token{
			TokenStart: t.position,
			TokenEnd:   t.position,
			TokenType:  EOF,
			Content:    "",
			Completed:  true,
		}
	}

	startPos := t.position
	char := t.buffer[t.position]

	switch char {
	case '{':
		t.position++
		t.expectingKey = true
		return Token{
			TokenStart: startPos,
			TokenEnd:   t.position,
			TokenType:  ObjectStart,
			Content:    singleChars['{'],
			Completed:  true,
		}
	case '}':
		t.position++
		t.expectingKey = false
		return Token{
			TokenStart: startPos,
			TokenEnd:   t.position,
			TokenType:  ObjectEnd,
			Content:    singleChars['}'],
			Completed:  true,
		}
	case '[':
		t.position++
		t.expectingKey = false
		return Token{
			TokenStart: startPos,
			TokenEnd:   t.position,
			TokenType:  ArrayStart,
			Content:    singleChars['['],
			Completed:  true,
		}
	case ']':
		t.position++
		return Token{
			TokenStart: startPos,
			TokenEnd:   t.position,
			TokenType:  ArrayEnd,
			Content:    singleChars[']'],
			Completed:  true,
		}
	case ':':
		t.position++
		t.expectingKey = false
		return Token{
			TokenStart: startPos,
			TokenEnd:   t.position,
			TokenType:  Colon,
			Content:    singleChars[':'],
			Completed:  true,
		}
	case ',':
		t.position++
		t.expectingKey = true // After comma in object, expect key
		return Token{
			TokenStart: startPos,
			TokenEnd:   t.position,
			TokenType:  Comma,
			Content:    singleChars[','],
			Completed:  true,
		}
	case '"':
		return t.parseString(startPos)
	case 't', 'f':
		return t.parseBool(startPos)
	case 'n':
		return t.parseNull(startPos)
	default:
		if char == '-' || (char >= '0' && char <= '9') {
			return t.parseNumber(startPos)
		}
		// Invalid character
		t.position++
		return Token{
			TokenStart: startPos,
			TokenEnd:   t.position,
			TokenType:  Invalid,
			Content:    singleChars[char],
			Completed:  true,
		}
	}
}

// continueToken continues parsing an incomplete token
func (t *StreamJSONTokenizer) continueToken() Token {
	if t.lastToken == nil {
		return Token{TokenType: Invalid, Completed: true}
	}

	switch t.lastToken.TokenType {
	case String, ObjectKey:
		return t.continueString(*t.lastToken)
	case Number:
		return t.continueNumber(*t.lastToken)
	case Bool:
		return t.continueBool(*t.lastToken)
	case Null:
		return t.continueNull(*t.lastToken)
	default:
		return *t.lastToken
	}
}

// skipWhitespace skips whitespace characters using fast byte comparison
func (t *StreamJSONTokenizer) skipWhitespace() {
	for t.position < len(t.buffer) {
		char := t.buffer[t.position]
		// Fast byte-level whitespace check for common cases
		if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
			t.position++
		} else {
			break
		}
	}
}

// buildString efficiently builds a string from buffer slice
func (t *StreamJSONTokenizer) buildString(start, end int) string {
	if start >= end {
		return ""
	}
	// Reset and reuse the builder for efficiency
	t.contentBuilder.Reset()
	t.contentBuilder.Write(t.buffer[start:end])
	return t.contentBuilder.String()
}

// parseString parses a string token
func (t *StreamJSONTokenizer) parseString(startPos int) Token {
	t.position++ // Skip opening quote
	contentStart := startPos

	for t.position < len(t.buffer) {
		char := t.buffer[t.position]
		t.position++

		if t.escapeNext {
			t.escapeNext = false
			continue
		}

		if char == '\\' {
			t.escapeNext = true
			continue
		}

		if char == '"' {
			// String is complete
			tokenType := String
			if t.expectingKey {
				tokenType = ObjectKey
			}
			return Token{
				TokenStart: startPos,
				TokenEnd:   t.position,
				TokenType:  tokenType,
				Content:    t.buildString(contentStart, t.position),
				Completed:  true,
			}
		}
	}

	// String is incomplete
	tokenType := String
	if t.expectingKey {
		tokenType = ObjectKey
	}
	token := Token{
		TokenStart: startPos,
		TokenEnd:   t.position,
		TokenType:  tokenType,
		Content:    t.buildString(contentStart, t.position),
		Completed:  false,
	}
	t.lastToken = &token
	return token
}

// continueString continues parsing an incomplete string
func (t *StreamJSONTokenizer) continueString(token Token) Token {
	for t.position < len(t.buffer) {
		char := t.buffer[t.position]
		t.position++

		if t.escapeNext {
			t.escapeNext = false
			continue
		}

		if char == '\\' {
			t.escapeNext = true
			continue
		}

		if char == '"' {
			// String is now complete
			return Token{
				TokenStart: token.TokenStart,
				TokenEnd:   t.position,
				TokenType:  token.TokenType,
				Content:    t.buildString(token.TokenStart, t.position),
				Completed:  true,
			}
		}
	}

	// Still incomplete
	return Token{
		TokenStart: token.TokenStart,
		TokenEnd:   t.position,
		TokenType:  token.TokenType,
		Content:    t.buildString(token.TokenStart, t.position),
		Completed:  false,
	}
}

// parseNumber parses a number token
func (t *StreamJSONTokenizer) parseNumber(startPos int) Token {
	// Handle negative sign
	if t.position < len(t.buffer) && t.buffer[t.position] == '-' {
		t.position++
	}

	// Parse digits and number characters
	for t.position < len(t.buffer) {
		char := t.buffer[t.position]
		if isNumberChar(char) {
			t.position++
		} else {
			break
		}
	}

	// Check if number is complete
	completed := false
	if t.position < len(t.buffer) {
		// If there's more content, check if next char would continue the number
		nextChar := t.buffer[t.position]
		if !isNumberChar(nextChar) {
			// Next char is not a number char, so this number is complete
			completed = true
		}
	}
	// If we're at the end of content, the number is incomplete until terminated by another token

	token := Token{
		TokenStart: startPos,
		TokenEnd:   t.position,
		TokenType:  Number,
		Content:    t.buildString(startPos, t.position),
		Completed:  completed,
	}

	if !completed {
		t.lastToken = &token
	}

	return token
}

// continueNumber continues parsing an incomplete number
func (t *StreamJSONTokenizer) continueNumber(token Token) Token {
	for t.position < len(t.buffer) {
		char := t.buffer[t.position]
		if isNumberChar(char) {
			t.position++
		} else {
			break
		}
	}

	// Check if number is now complete
	completed := false
	if t.position < len(t.buffer) {
		// If there's more content, check if next char would continue the number
		nextChar := t.buffer[t.position]
		if !isNumberChar(nextChar) {
			// Next char is not a number char, so this number is complete
			completed = true
		}
	}
	// If we're at the end of content, the number might still be incomplete

	return Token{
		TokenStart: token.TokenStart,
		TokenEnd:   t.position,
		TokenType:  Number,
		Content:    t.buildString(token.TokenStart, t.position),
		Completed:  completed,
	}
}

// isNumberChar checks if character can be part of a number
func isNumberChar(char byte) bool {
	return (char >= '0' && char <= '9') || char == '.' || char == 'e' || char == 'E' || char == '+' || char == '-'
}

// parseBool parses a boolean token (true/false)
func (t *StreamJSONTokenizer) parseBool(startPos int) Token {
	// Determine which boolean we're parsing
	var expected []byte
	if t.buffer[t.position] == 't' {
		expected = trueBytes
	} else {
		expected = falseBytes
	}

	// Check if we can match the full expected word
	for i := 0; i < len(expected) && t.position < len(t.buffer); i++ {
		if t.buffer[t.position] != expected[i] {
			// Invalid boolean
			t.position++
			return Token{
				TokenStart: startPos,
				TokenEnd:   t.position,
				TokenType:  Invalid,
				Content:    t.buildString(startPos, t.position),
				Completed:  true,
			}
		}
		t.position++
	}

	// Check if we've read the complete word
	if t.position-startPos == len(expected) {
		// Check if next character is a valid terminator
		if t.position < len(t.buffer) {
			nextChar := t.buffer[t.position]
			if isLetter(nextChar) {
				// Still part of an invalid word
				for t.position < len(t.buffer) && isLetter(t.buffer[t.position]) {
					t.position++
				}
				return Token{
					TokenStart: startPos,
					TokenEnd:   t.position,
					TokenType:  Invalid,
					Content:    t.buildString(startPos, t.position),
					Completed:  true,
				}
			}
		}

		// Complete and valid boolean
		return Token{
			TokenStart: startPos,
			TokenEnd:   t.position,
			TokenType:  Bool,
			Content:    t.buildString(startPos, t.position),
			Completed:  true,
		}
	}

	// Incomplete boolean
	token := Token{
		TokenStart: startPos,
		TokenEnd:   t.position,
		TokenType:  Bool,
		Content:    t.buildString(startPos, t.position),
		Completed:  false,
	}
	t.lastToken = &token
	return token
}

// continueBool continues parsing an incomplete boolean
func (t *StreamJSONTokenizer) continueBool(token Token) Token {
	tokenLen := token.TokenEnd - token.TokenStart

	// Determine which boolean we're parsing based on current content
	var expected []byte
	if tokenLen > 0 && t.buffer[token.TokenStart] == 't' {
		expected = trueBytes
	} else {
		expected = falseBytes
	}

	// Continue matching from where we left off
	for i := tokenLen; i < len(expected) && t.position < len(t.buffer); i++ {
		if t.buffer[t.position] != expected[i] {
			// Invalid boolean
			for t.position < len(t.buffer) && isLetter(t.buffer[t.position]) {
				t.position++
			}
			return Token{
				TokenStart: token.TokenStart,
				TokenEnd:   t.position,
				TokenType:  Invalid,
				Content:    t.buildString(token.TokenStart, t.position),
				Completed:  true,
			}
		}
		t.position++
	}

	// Check if complete
	if t.position-token.TokenStart == len(expected) {
		// Check terminator
		if t.position < len(t.buffer) && isLetter(t.buffer[t.position]) {
			// Invalid - continues with more letters
			for t.position < len(t.buffer) && isLetter(t.buffer[t.position]) {
				t.position++
			}
			return Token{
				TokenStart: token.TokenStart,
				TokenEnd:   t.position,
				TokenType:  Invalid,
				Content:    t.buildString(token.TokenStart, t.position),
				Completed:  true,
			}
		}

		return Token{
			TokenStart: token.TokenStart,
			TokenEnd:   t.position,
			TokenType:  Bool,
			Content:    t.buildString(token.TokenStart, t.position),
			Completed:  true,
		}
	}

	return Token{
		TokenStart: token.TokenStart,
		TokenEnd:   t.position,
		TokenType:  Bool,
		Content:    t.buildString(token.TokenStart, t.position),
		Completed:  false,
	}
}

// parseNull parses a null token
func (t *StreamJSONTokenizer) parseNull(startPos int) Token {
	// Check if we can match "null"
	for i := 0; i < len(nullBytes) && t.position < len(t.buffer); i++ {
		if t.buffer[t.position] != nullBytes[i] {
			// Invalid null
			t.position++
			return Token{
				TokenStart: startPos,
				TokenEnd:   t.position,
				TokenType:  Invalid,
				Content:    t.buildString(startPos, t.position),
				Completed:  true,
			}
		}
		t.position++
	}

	// Check if we've read the complete word
	if t.position-startPos == len(nullBytes) {
		// Check if next character is a valid terminator
		if t.position < len(t.buffer) {
			nextChar := t.buffer[t.position]
			if isLetter(nextChar) {
				// Still part of an invalid word
				for t.position < len(t.buffer) && isLetter(t.buffer[t.position]) {
					t.position++
				}
				return Token{
					TokenStart: startPos,
					TokenEnd:   t.position,
					TokenType:  Invalid,
					Content:    t.buildString(startPos, t.position),
					Completed:  true,
				}
			}
		}

		// Complete and valid null
		return Token{
			TokenStart: startPos,
			TokenEnd:   t.position,
			TokenType:  Null,
			Content:    t.buildString(startPos, t.position),
			Completed:  true,
		}
	}

	// Incomplete null
	token := Token{
		TokenStart: startPos,
		TokenEnd:   t.position,
		TokenType:  Null,
		Content:    t.buildString(startPos, t.position),
		Completed:  false,
	}
	t.lastToken = &token
	return token
}

// continueNull continues parsing an incomplete null
func (t *StreamJSONTokenizer) continueNull(token Token) Token {
	tokenLen := token.TokenEnd - token.TokenStart

	// Continue matching from where we left off
	for i := tokenLen; i < len(nullBytes) && t.position < len(t.buffer); i++ {
		if t.buffer[t.position] != nullBytes[i] {
			// Invalid null
			for t.position < len(t.buffer) && isLetter(t.buffer[t.position]) {
				t.position++
			}
			return Token{
				TokenStart: token.TokenStart,
				TokenEnd:   t.position,
				TokenType:  Invalid,
				Content:    t.buildString(token.TokenStart, t.position),
				Completed:  true,
			}
		}
		t.position++
	}

	// Check if complete
	if t.position-token.TokenStart == len(nullBytes) {
		// Check terminator
		if t.position < len(t.buffer) && isLetter(t.buffer[t.position]) {
			// Invalid - continues with more letters
			for t.position < len(t.buffer) && isLetter(t.buffer[t.position]) {
				t.position++
			}
			return Token{
				TokenStart: token.TokenStart,
				TokenEnd:   t.position,
				TokenType:  Invalid,
				Content:    t.buildString(token.TokenStart, t.position),
				Completed:  true,
			}
		}

		return Token{
			TokenStart: token.TokenStart,
			TokenEnd:   t.position,
			TokenType:  Null,
			Content:    t.buildString(token.TokenStart, t.position),
			Completed:  true,
		}
	}

	return Token{
		TokenStart: token.TokenStart,
		TokenEnd:   t.position,
		TokenType:  Null,
		Content:    t.buildString(token.TokenStart, t.position),
		Completed:  false,
	}
}

// isLetter checks if character is a letter using fast byte comparison
func isLetter(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}
