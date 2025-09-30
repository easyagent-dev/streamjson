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

package streamjson

import (
	"testing"
)

func TestBasicTokenTypes(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()
	tokenizer.Append(`{"key":123,true,false,null,[]}`)

	// Test ObjectStart
	token := tokenizer.NextToken()
	if token.TokenType != ObjectStart || token.Content != "{" || !token.Completed {
		t.Errorf("Expected ObjectStart token, got %v", token)
	}

	// Test ObjectKey
	token = tokenizer.NextToken()
	if token.TokenType != ObjectKey || token.Content != `"key"` || !token.Completed {
		t.Errorf("Expected ObjectKey token, got %v", token)
	}

	// Test Colon
	token = tokenizer.NextToken()
	if token.TokenType != Colon || token.Content != ":" || !token.Completed {
		t.Errorf("Expected Colon token, got %v", token)
	}

	// Test Number
	token = tokenizer.NextToken()
	if token.TokenType != Number || token.Content != "123" || !token.Completed {
		t.Errorf("Expected Number token, got %v", token)
	}

	// Test Comma
	token = tokenizer.NextToken()
	if token.TokenType != Comma || token.Content != "," || !token.Completed {
		t.Errorf("Expected Comma token, got %v", token)
	}

	// Test Bool (true)
	token = tokenizer.NextToken()
	if token.TokenType != Bool || token.Content != "true" || !token.Completed {
		t.Errorf("Expected Bool token, got %v", token)
	}

	// Test Comma
	token = tokenizer.NextToken()
	if token.TokenType != Comma || token.Content != "," || !token.Completed {
		t.Errorf("Expected Comma token, got %v", token)
	}

	// Test Bool (false)
	token = tokenizer.NextToken()
	if token.TokenType != Bool || token.Content != "false" || !token.Completed {
		t.Errorf("Expected Bool token, got %v", token)
	}

	// Test Comma
	token = tokenizer.NextToken()
	if token.TokenType != Comma || token.Content != "," || !token.Completed {
		t.Errorf("Expected Comma token, got %v", token)
	}

	// Test Null
	token = tokenizer.NextToken()
	if token.TokenType != Null || token.Content != "null" || !token.Completed {
		t.Errorf("Expected Null token, got %v", token)
	}

	// Test Comma
	token = tokenizer.NextToken()
	if token.TokenType != Comma || token.Content != "," || !token.Completed {
		t.Errorf("Expected Comma token, got %v", token)
	}

	// Test ArrayStart
	token = tokenizer.NextToken()
	if token.TokenType != ArrayStart || token.Content != "[" || !token.Completed {
		t.Errorf("Expected ArrayStart token, got %v", token)
	}

	// Test ArrayEnd
	token = tokenizer.NextToken()
	if token.TokenType != ArrayEnd || token.Content != "]" || !token.Completed {
		t.Errorf("Expected ArrayEnd token, got %v", token)
	}

	// Test ObjectEnd
	token = tokenizer.NextToken()
	if token.TokenType != ObjectEnd || token.Content != "}" || !token.Completed {
		t.Errorf("Expected ObjectEnd token, got %v", token)
	}

	// Test EOF
	token = tokenizer.NextToken()
	if token.TokenType != EOF || !token.Completed {
		t.Errorf("Expected EOF token, got %v", token)
	}
}

func TestPartialStringHandling(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()

	// Test the exact scenario from the requirements:
	// 1. {"name":"hello
	// 2. world
	// 3. "}
	tokenizer.Append(`{"name":"hello `)

	// Get ObjectStart
	token := tokenizer.NextToken()
	if token.TokenType != ObjectStart {
		t.Errorf("Expected ObjectStart, got %v", token)
	}

	// Get ObjectKey
	token = tokenizer.NextToken()
	if token.TokenType != ObjectKey || token.Content != `"name"` {
		t.Errorf("Expected ObjectKey 'name', got %v", token)
	}

	// Get Colon
	token = tokenizer.NextToken()
	if token.TokenType != Colon {
		t.Errorf("Expected Colon, got %v", token)
	}

	// Get incomplete string
	token = tokenizer.NextToken()
	if token.TokenType != String || token.Content != `"hello ` || token.Completed {
		t.Errorf("Expected incomplete String 'hello ', got %v", token)
	}

	// Append more content
	tokenizer.Append(`world`)

	// Continue parsing the string
	token = tokenizer.NextToken()
	if token.TokenType != String || token.Content != `"hello world` || token.Completed {
		t.Errorf("Expected incomplete String 'hello world', got %v", token)
	}

	// Append final part
	tokenizer.Append(`"}`)

	// Complete the string
	token = tokenizer.NextToken()
	if token.TokenType != String || token.Content != `"hello world"` || !token.Completed {
		t.Errorf("Expected complete String 'hello world', got %v", token)
	}

	// Get ObjectEnd
	token = tokenizer.NextToken()
	if token.TokenType != ObjectEnd {
		t.Errorf("Expected ObjectEnd, got %v", token)
	}
}

func TestPartialNumberHandling(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()

	// Test partial number
	tokenizer.Append(`12`)
	token := tokenizer.NextToken()
	if token.TokenType != Number || token.Content != "12" || token.Completed {
		t.Errorf("Expected incomplete Number '12', got %v", token)
	}

	// Continue with decimal
	tokenizer.Append(`.34`)
	token = tokenizer.NextToken()
	if token.TokenType != Number || token.Content != "12.34" || token.Completed {
		t.Errorf("Expected incomplete Number '12.34', got %v", token)
	}

	// Complete with exponent
	tokenizer.Append(`e-5 `)
	token = tokenizer.NextToken()
	if token.TokenType != Number || token.Content != "12.34e-5" || !token.Completed {
		t.Errorf("Expected complete Number '12.34e-5', got %v", token)
	}
}

func TestPartialBooleanHandling(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()

	// Test partial "true"
	tokenizer.Append(`tr`)
	token := tokenizer.NextToken()
	if token.TokenType != Bool || token.Content != "tr" || token.Completed {
		t.Errorf("Expected incomplete Bool 'tr', got %v", token)
	}

	tokenizer.Append(`ue `)
	token = tokenizer.NextToken()
	if token.TokenType != Bool || token.Content != "true" || !token.Completed {
		t.Errorf("Expected complete Bool 'true', got %v", token)
	}

	// Test partial "false"
	tokenizer.Append(`fal`)
	token = tokenizer.NextToken()
	if token.TokenType != Bool || token.Content != "fal" || token.Completed {
		t.Errorf("Expected incomplete Bool 'fal', got %v", token)
	}

	tokenizer.Append(`se,`)
	token = tokenizer.NextToken()
	if token.TokenType != Bool || token.Content != "false" || !token.Completed {
		t.Errorf("Expected complete Bool 'false', got %v", token)
	}
}

func TestPartialNullHandling(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()

	// Test partial "null"
	tokenizer.Append(`nu`)
	token := tokenizer.NextToken()
	if token.TokenType != Null || token.Content != "nu" || token.Completed {
		t.Errorf("Expected incomplete Null 'nu', got %v", token)
	}

	tokenizer.Append(`ll}`)
	token = tokenizer.NextToken()
	if token.TokenType != Null || token.Content != "null" || !token.Completed {
		t.Errorf("Expected complete Null 'null', got %v", token)
	}
}

func TestStringWithEscapes(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()
	tokenizer.Append(`"hello \"world\" test"`)

	token := tokenizer.NextToken()
	if token.TokenType != String || token.Content != `"hello \"world\" test"` || !token.Completed {
		t.Errorf("Expected escaped string, got %v", token)
	}
}

func TestPartialStringWithEscapes(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()

	// Partial string with escape at the end
	tokenizer.Append(`"hello \`)
	token := tokenizer.NextToken()
	if token.TokenType != String || token.Content != `"hello \` || token.Completed {
		t.Errorf("Expected incomplete escaped string, got %v", token)
	}

	// Complete the escape
	tokenizer.Append(`"world"`)
	token = tokenizer.NextToken()
	if token.TokenType != String || token.Content != `"hello \"world"` || !token.Completed {
		t.Errorf("Expected complete escaped string, got %v", token)
	}
}

func TestNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123", "123"},
		{"-456", "-456"},
		{"12.34", "12.34"},
		{"-12.34", "-12.34"},
		{"1.23e10", "1.23e10"},
		{"1.23E-10", "1.23E-10"},
		{"-1.23e+10", "-1.23e+10"},
	}

	for _, test := range tests {
		tokenizer := NewStreamJSONTokenizer()
		tokenizer.Append(test.input + " ")

		token := tokenizer.NextToken()
		if token.TokenType != Number || token.Content != test.expected || !token.Completed {
			t.Errorf("Input: %s, Expected Number '%s', got %v", test.input, test.expected, token)
		}
	}
}

func TestWhitespaceHandling(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()
	tokenizer.Append(`  {  "key"  :  123  }  `)

	// Should skip whitespace and get ObjectStart
	token := tokenizer.NextToken()
	if token.TokenType != ObjectStart {
		t.Errorf("Expected ObjectStart after whitespace, got %v", token)
	}

	// Should skip whitespace and get ObjectKey
	token = tokenizer.NextToken()
	if token.TokenType != ObjectKey || token.Content != `"key"` {
		t.Errorf("Expected ObjectKey after whitespace, got %v", token)
	}
}

func TestInvalidTokens(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()
	tokenizer.Append(`@`)

	token := tokenizer.NextToken()
	if token.TokenType != Invalid || token.Content != "@" || !token.Completed {
		t.Errorf("Expected Invalid token, got %v", token)
	}
}

func TestComplexPartialJSON(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()

	// Simulate streaming JSON: {"users":[{"name":"John","age":30}]}
	tokenizer.Append(`{"users":[{"name":"Jo`)

	// Get tokens up to partial string
	tokens := []Token{}
	for {
		token := tokenizer.NextToken()
		tokens = append(tokens, token)
		if token.TokenType == EOF || (!token.Completed && token.TokenType == String) {
			break
		}
	}

	// Check we got the expected tokens
	expectedTypes := []TokenType{ObjectStart, ObjectKey, Colon, ArrayStart, ObjectStart, ObjectKey, Colon, String}
	if len(tokens) != len(expectedTypes) {
		t.Errorf("Expected %d tokens, got %d", len(expectedTypes), len(tokens))
	}

	for i, expectedType := range expectedTypes {
		if i < len(tokens) && tokens[i].TokenType != expectedType {
			t.Errorf("Token %d: expected %v, got %v", i, expectedType, tokens[i].TokenType)
		}
	}

	// Last token should be incomplete string
	lastToken := tokens[len(tokens)-1]
	if lastToken.TokenType != String || lastToken.Content != `"Jo` || lastToken.Completed {
		t.Errorf("Expected incomplete string, got %v", lastToken)
	}

	// Continue with more content
	tokenizer.Append(`hn","age":30}]}`)

	// Get remaining tokens
	for {
		token := tokenizer.NextToken()
		if token.TokenType == EOF {
			break
		}
		if !token.Completed {
			t.Errorf("Got incomplete token when all should be complete: %v", token)
		}
	}
}

func TestPositionTracking(t *testing.T) {
	tokenizer := NewStreamJSONTokenizer()
	tokenizer.Append(`{"key":123}`)

	token := tokenizer.NextToken() // {
	if token.TokenStart != 0 || token.TokenEnd != 1 {
		t.Errorf("ObjectStart position: expected 0-1, got %d-%d", token.TokenStart, token.TokenEnd)
	}

	token = tokenizer.NextToken() // "key"
	if token.TokenStart != 1 || token.TokenEnd != 6 {
		t.Errorf("ObjectKey position: expected 1-6, got %d-%d", token.TokenStart, token.TokenEnd)
	}

	token = tokenizer.NextToken() // :
	if token.TokenStart != 6 || token.TokenEnd != 7 {
		t.Errorf("Colon position: expected 6-7, got %d-%d", token.TokenStart, token.TokenEnd)
	}

	token = tokenizer.NextToken() // 123
	if token.TokenStart != 7 || token.TokenEnd != 10 {
		t.Errorf("Number position: expected 7-10, got %d-%d", token.TokenStart, token.TokenEnd)
	}

	token = tokenizer.NextToken() // }
	if token.TokenStart != 10 || token.TokenEnd != 11 {
		t.Errorf("ObjectEnd position: expected 10-11, got %d-%d", token.TokenStart, token.TokenEnd)
	}
}
