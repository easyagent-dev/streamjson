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
	"strconv"
	"sync"
)

// NodeType represents the type of AST node
type NodeType int

const (
	ObjectNode NodeType = iota
	ArrayNode
	ValueNode
)

// Node represents a node in the AST tree
type Node struct {
	Type      NodeType
	Value     interface{}
	Children  map[string]*Node // For objects
	Array     []*Node          // For arrays
	Completed bool             // Whether this node is complete
	Parent    *Node            // Reference to parent node
}

// Object pools for memory reuse
var (
	nodePool = sync.Pool{
		New: func() interface{} {
			return &Node{}
		},
	}
	stackFramePool = sync.Pool{
		New: func() interface{} {
			return &StackFrame{}
		},
	}
)

// NewNode creates a new AST node with object pooling
func NewNode(nodeType NodeType) *Node {
	node := nodePool.Get().(*Node)

	// Reset the node
	node.Type = nodeType
	node.Value = nil
	node.Completed = false
	node.Parent = nil

	// Clear existing children/array but reuse maps/slices when possible
	if nodeType == ObjectNode {
		if node.Children == nil {
			node.Children = make(map[string]*Node, 8) // Pre-allocate reasonable size
		} else {
			// Clear existing map but keep capacity
			for k := range node.Children {
				delete(node.Children, k)
			}
		}
		node.Array = nil
	} else if nodeType == ArrayNode {
		if node.Array == nil {
			node.Array = make([]*Node, 0, 8) // Pre-allocate reasonable capacity
		} else {
			// Reset slice but keep capacity
			node.Array = node.Array[:0]
		}
		node.Children = nil
	} else {
		node.Children = nil
		node.Array = nil
	}

	return node
}

// ReleaseNode returns a node to the pool
func ReleaseNode(node *Node) {
	if node == nil {
		return
	}

	// Recursively release child nodes
	if node.Children != nil {
		for _, child := range node.Children {
			ReleaseNode(child)
		}
	}
	if node.Array != nil {
		for _, child := range node.Array {
			ReleaseNode(child)
		}
	}

	nodePool.Put(node)
}

// newStackFrame creates a new stack frame with pooling
func newStackFrame() *StackFrame {
	frame := stackFramePool.Get().(*StackFrame)
	// Reset fields
	frame.Node = nil
	frame.CurrentKey = ""
	frame.ExpectingKey = false
	frame.ExpectingValue = false
	return frame
}

// releaseStackFrame returns a stack frame to the pool
func releaseStackFrame(frame *StackFrame) {
	if frame != nil {
		stackFramePool.Put(frame)
	}
}

// StackFrame represents a frame in the parsing stack
type StackFrame struct {
	Node           *Node
	CurrentKey     string // For objects, the current key being parsed
	ExpectingKey   bool   // For objects, whether we're expecting a key next
	ExpectingValue bool   // Whether we're expecting a value next
}

// StreamJSONParser implements a streaming JSON parser with AST building
type StreamJSONParser struct {
	tokenizer *StreamJSONTokenizer
	root      *Node
	stack     []*StackFrame
	started   bool
}

// NewStreamJSONParser creates a new streaming JSON parser
func NewStreamJSONParser() *StreamJSONParser {
	return &StreamJSONParser{
		tokenizer: NewStreamJSONTokenizer(),
		stack:     make([]*StackFrame, 0, 16), // Pre-allocate reasonable stack capacity
		started:   false,
	}
}

// Append adds more content to the parser and processes tokens
func (p *StreamJSONParser) Append(content string) {
	p.tokenizer.Append(content)
	p.processTokens()
}

// processTokens processes available tokens and builds the AST
func (p *StreamJSONParser) processTokens() {
	// Keep processing until no more complete tokens are available
	for {
		token := p.tokenizer.NextToken()

		// Handle EOF or invalid tokens
		if token.TokenType == EOF {
			break
		}

		if token.TokenType == Invalid {
			continue // Tolerate errors as required
		}

		// If we haven't started, we need ObjectStart or ArrayStart
		if !p.started {
			if token.TokenType == ObjectStart {
				p.root = NewNode(ObjectNode)
				frame := newStackFrame()
				frame.Node = p.root
				frame.ExpectingKey = true
				p.stack = append(p.stack, frame)
				p.started = true
			} else if token.TokenType == ArrayStart {
				p.root = NewNode(ArrayNode)
				frame := newStackFrame()
				frame.Node = p.root
				frame.ExpectingValue = true
				p.stack = append(p.stack, frame)
				p.started = true
			}
			// Tolerate other tokens until we find a valid start
			continue
		}

		// Process both completed and incomplete tokens
		if token.Completed {
			p.processCompleteToken(token)
		} else {
			// Handle incomplete tokens for partial access
			p.processIncompleteToken(token)
			break // Break after handling incomplete token
		}
	}
}

// processIncompleteToken processes an incomplete token for partial access
func (p *StreamJSONParser) processIncompleteToken(token Token) {
	if len(p.stack) == 0 {
		return // No active parsing context
	}

	currentFrame := p.stack[len(p.stack)-1]

	// Handle incomplete strings for partial access
	if token.TokenType == String && currentFrame.Node.Type == ObjectNode && currentFrame.CurrentKey != "" {
		content := token.Content
		if len(content) >= 1 && content[0] == '"' {
			partialValue := content[1:] // Remove opening quote

			// Provide partial access for any incomplete string
			valueNode := NewNode(ValueNode)
			valueNode.Value = partialValue
			valueNode.Completed = false // Mark as incomplete
			valueNode.Parent = currentFrame.Node

			// Store the partial value in the AST
			currentFrame.Node.Children[currentFrame.CurrentKey] = valueNode
		}
	}
}

// processCompleteToken processes a complete token
func (p *StreamJSONParser) processCompleteToken(token Token) {
	if len(p.stack) == 0 {
		return // No active parsing context
	}

	currentFrame := p.stack[len(p.stack)-1]

	switch token.TokenType {
	case ObjectStart:
		p.handleObjectStart(currentFrame)

	case ArrayStart:
		p.handleArrayStart(currentFrame)

	case ObjectEnd:
		p.handleObjectEnd()

	case ArrayEnd:
		p.handleArrayEnd()

	case ObjectKey:
		p.handleObjectKey(token, currentFrame)

	case Colon:
		if currentFrame.Node.Type == ObjectNode {
			currentFrame.ExpectingKey = false
			currentFrame.ExpectingValue = true
		}

	case Comma:
		p.handleComma(currentFrame)

	case String, Number, Bool, Null:
		p.handleValue(token, currentFrame)
	}
}

// handleObjectStart handles the start of an object
func (p *StreamJSONParser) handleObjectStart(currentFrame *StackFrame) {
	newNode := NewNode(ObjectNode)
	newNode.Parent = currentFrame.Node

	if currentFrame.Node.Type == ObjectNode && currentFrame.CurrentKey != "" {
		currentFrame.Node.Children[currentFrame.CurrentKey] = newNode
		currentFrame.CurrentKey = ""
	} else if currentFrame.Node.Type == ArrayNode {
		currentFrame.Node.Array = append(currentFrame.Node.Array, newNode)
	}

	frame := newStackFrame()
	frame.Node = newNode
	frame.ExpectingKey = true
	p.stack = append(p.stack, frame)
}

// handleArrayStart handles the start of an array
func (p *StreamJSONParser) handleArrayStart(currentFrame *StackFrame) {
	newNode := NewNode(ArrayNode)
	newNode.Parent = currentFrame.Node

	if currentFrame.Node.Type == ObjectNode && currentFrame.CurrentKey != "" {
		currentFrame.Node.Children[currentFrame.CurrentKey] = newNode
		currentFrame.CurrentKey = ""
	} else if currentFrame.Node.Type == ArrayNode {
		currentFrame.Node.Array = append(currentFrame.Node.Array, newNode)
	}

	frame := newStackFrame()
	frame.Node = newNode
	frame.ExpectingValue = true
	p.stack = append(p.stack, frame)
}

// handleObjectEnd handles the end of an object
func (p *StreamJSONParser) handleObjectEnd() {
	if len(p.stack) > 0 {
		currentFrame := p.stack[len(p.stack)-1]
		currentFrame.Node.Completed = true
		releaseStackFrame(currentFrame)
		p.stack = p.stack[:len(p.stack)-1]

		// Update parent frame state
		if len(p.stack) > 0 {
			parentFrame := p.stack[len(p.stack)-1]
			if parentFrame.Node.Type == ObjectNode {
				parentFrame.ExpectingKey = false
				parentFrame.ExpectingValue = false
			} else if parentFrame.Node.Type == ArrayNode {
				parentFrame.ExpectingValue = false
			}
		}
	}
}

// handleArrayEnd handles the end of an array
func (p *StreamJSONParser) handleArrayEnd() {
	if len(p.stack) > 0 {
		currentFrame := p.stack[len(p.stack)-1]
		currentFrame.Node.Completed = true
		releaseStackFrame(currentFrame)
		p.stack = p.stack[:len(p.stack)-1]

		// Update parent frame state
		if len(p.stack) > 0 {
			parentFrame := p.stack[len(p.stack)-1]
			if parentFrame.Node.Type == ObjectNode {
				parentFrame.ExpectingKey = false
				parentFrame.ExpectingValue = false
			} else if parentFrame.Node.Type == ArrayNode {
				parentFrame.ExpectingValue = false
			}
		}
	}
}

// handleObjectKey handles an object key
func (p *StreamJSONParser) handleObjectKey(token Token, currentFrame *StackFrame) {
	if currentFrame.Node.Type == ObjectNode {
		// Extract the key from the quoted string efficiently
		content := token.Content
		if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
			currentFrame.CurrentKey = content[1 : len(content)-1]
		} else {
			currentFrame.CurrentKey = content
		}
		currentFrame.ExpectingKey = false
	}
}

// handleComma handles comma separators
func (p *StreamJSONParser) handleComma(currentFrame *StackFrame) {
	if currentFrame.Node.Type == ObjectNode {
		currentFrame.ExpectingKey = true
		currentFrame.ExpectingValue = false
		currentFrame.CurrentKey = ""
	} else if currentFrame.Node.Type == ArrayNode {
		currentFrame.ExpectingValue = true
	}
}

// handleValue handles value tokens (string, number, bool, null)
func (p *StreamJSONParser) handleValue(token Token, currentFrame *StackFrame) {
	valueNode := NewNode(ValueNode)
	valueNode.Value = p.parseTokenValue(token)
	valueNode.Completed = true
	valueNode.Parent = currentFrame.Node

	if currentFrame.Node.Type == ObjectNode && currentFrame.CurrentKey != "" {
		currentFrame.Node.Children[currentFrame.CurrentKey] = valueNode
		currentFrame.CurrentKey = ""
		currentFrame.ExpectingValue = false
	} else if currentFrame.Node.Type == ArrayNode {
		currentFrame.Node.Array = append(currentFrame.Node.Array, valueNode)
		currentFrame.ExpectingValue = false
	}
}

// parseTokenValue converts token content to appropriate Go value with optimized parsing
func (p *StreamJSONParser) parseTokenValue(token Token) interface{} {
	content := token.Content

	switch token.TokenType {
	case String:
		// Remove quotes from string content efficiently
		if len(content) >= 2 && content[0] == '"' && content[len(content)-1] == '"' {
			return content[1 : len(content)-1]
		}
		return content

	case Number:
		// Optimized number parsing - check for integer vs float efficiently
		hasDecimal := false
		hasExp := false

		for i := 0; i < len(content); i++ {
			c := content[i]
			if c == '.' {
				hasDecimal = true
				break
			} else if c == 'e' || c == 'E' {
				hasExp = true
				break
			}
		}

		if !hasDecimal && !hasExp {
			// Try integer parsing first for performance
			if val, err := strconv.ParseInt(content, 10, 64); err == nil {
				return val
			}
		}

		// Parse as float
		if val, err := strconv.ParseFloat(content, 64); err == nil {
			return val
		}

		return content // Fallback to string if parsing fails

	case Bool:
		// Optimized boolean check
		return len(content) == 4 && content[0] == 't' // "true" has length 4 and starts with 't'

	case Null:
		return nil

	default:
		return content
	}
}

// Get retrieves a value from the AST using a path of keys
func (p *StreamJSONParser) Get(keys ...string) interface{} {
	if p.root == nil || len(keys) == 0 {
		return nil
	}

	return p.getFromNode(p.root, keys)
}

// getFromNode recursively traverses the AST to find the value
func (p *StreamJSONParser) getFromNode(node *Node, keys []string) interface{} {
	if node == nil || len(keys) == 0 {
		if node != nil && node.Type == ValueNode {
			return node.Value
		}
		return node
	}

	key := keys[0]
	remainingKeys := keys[1:]

	switch node.Type {
	case ObjectNode:
		if child, exists := node.Children[key]; exists {
			if len(remainingKeys) == 0 {
				if child.Type == ValueNode {
					return child.Value
				}
				return child
			}
			return p.getFromNode(child, remainingKeys)
		}

	case ArrayNode:
		// Try to parse key as array index
		if index, err := strconv.Atoi(key); err == nil {
			if index >= 0 && index < len(node.Array) {
				child := node.Array[index]
				if len(remainingKeys) == 0 {
					if child.Type == ValueNode {
						return child.Value
					}
					return child
				}
				return p.getFromNode(child, remainingKeys)
			}
		}
	}

	return nil
}

// IsCompleted returns true if the parsing stack is empty (all structures closed)
func (p *StreamJSONParser) IsCompleted() bool {
	return len(p.stack) == 0 && p.started
}

// GetRoot returns the root node of the AST
func (p *StreamJSONParser) GetRoot() *Node {
	return p.root
}
