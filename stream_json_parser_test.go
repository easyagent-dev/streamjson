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

func TestStreamJSONParserBasic(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test complete JSON in one go
	parser.Append(`{"name":"John","age":30}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	name := parser.Get("name")
	if name != "John" {
		t.Errorf("Expected name to be 'John', got %v", name)
	}

	age := parser.Get("age")
	if age != int64(30) {
		t.Errorf("Expected age to be 30, got %v", age)
	}
}

func TestStreamJSONParserArray(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test array parsing
	parser.Append(`[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	alice := parser.Get("0", "name")
	if alice != "Alice" {
		t.Errorf("Expected first name to be 'Alice', got %v", alice)
	}

	bob := parser.Get("1", "name")
	if bob != "Bob" {
		t.Errorf("Expected second name to be 'Bob', got %v", bob)
	}
}

func TestStreamJSONParserErrorTolerance(t *testing.T) {
	parser := NewStreamJSONParser()

	// Start with some invalid tokens (should be tolerated)
	parser.Append(`invalid text {"valid": true}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	valid := parser.Get("valid")
	if valid != true {
		t.Errorf("Expected valid to be true, got %v", valid)
	}
}

func TestStreamJSONParserIncremental(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test incremental parsing with complete tokens only
	parser.Append(`{`)
	if parser.IsCompleted() {
		t.Errorf("Expected parser to not be completed after opening brace")
	}

	parser.Append(`"name"`)
	parser.Append(`:`)
	parser.Append(`"John"`)
	name := parser.Get("name")
	if name != "John" {
		t.Errorf("Expected name to be 'John', got %v", name)
	}

	parser.Append(`}`)
	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed after closing brace")
	}
}

func TestStreamJSONParserMultiAppendString(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test multi-append for a single string value
	parser.Append(`{"message":`)

	// Should be nil since string hasn't started yet
	message := parser.Get("message")
	if message != nil {
		t.Errorf("Expected message to be nil before string starts, got %v", message)
	}

	parser.Append(`"Hel`)
	// Should still be nil since string is incomplete
	message = parser.Get("message")
	if message != "Hel" {
		t.Errorf("Expected message to be 'Hel' for incomplete string, got %v", message)
	}

	parser.Append(`lo wor`)
	// Should still be nil since string is incomplete
	message = parser.Get("message")
	if message != "Hello wor" {
		t.Errorf("Expected message to be 'Hello wor' for incomplete string, got %v", message)
	}

	parser.Append(`ld"`)
	// Now string should be complete and available
	message = parser.Get("message")
	if message != "Hello world" {
		t.Errorf("Expected message to be 'Hello world' after completion, got %v", message)
	}

	parser.Append(`}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	message = parser.Get("message")
	if message != "Hello world" {
		t.Errorf("Expected message to be 'Hello world', got %v", message)
	}
}

func TestStreamJSONParserMultiAppendNumber(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test multi-append for a number value with partial retrieval checks
	parser.Append(`{"value":`)

	// Should be nil since number hasn't started yet
	value := parser.Get("value")
	if value != nil {
		t.Errorf("Expected value to be nil before number starts, got %v", value)
	}

	parser.Append(`12`)
	// Should still be nil since number is incomplete
	value = parser.Get("value")
	if value != nil {
		t.Errorf("Expected value to be nil for incomplete number, got %v", value)
	}

	parser.Append(`3.`)
	// Should still be nil since number is incomplete
	value = parser.Get("value")
	if value != nil {
		t.Errorf("Expected value to be nil for incomplete number, got %v", value)
	}

	parser.Append(`45`)
	// Number might still be incomplete until we know it's terminated
	parser.Append(`}`)

	// Now number should be complete and available
	value = parser.Get("value")
	if value != 123.45 {
		t.Errorf("Expected value to be 123.45, got %v", value)
	}

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}
}

func TestStreamJSONParserGetObject(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test getting an object node - should return map[string]interface{} instead of *Node
	parser.Append(`{"user":{"name":"Alice","age":25},"status":"active"}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	// Get the entire user object
	user := parser.Get("user")
	userMap, ok := user.(map[string]interface{})
	if !ok {
		t.Errorf("Expected user to be map[string]interface{}, got %T", user)
	}

	// Verify the map contains the correct data
	if userMap["name"] != "Alice" {
		t.Errorf("Expected name to be 'Alice', got %v", userMap["name"])
	}

	if userMap["age"] != int64(25) {
		t.Errorf("Expected age to be 25, got %v", userMap["age"])
	}
}

func TestStreamJSONParserGetArray(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test getting an array node - should return []interface{} instead of *Node
	parser.Append(`{"items":[1,2,3],"count":3}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	// Get the entire items array
	items := parser.Get("items")
	itemsSlice, ok := items.([]interface{})
	if !ok {
		t.Errorf("Expected items to be []interface{}, got %T", items)
	}

	// Verify the slice contains the correct data
	if len(itemsSlice) != 3 {
		t.Errorf("Expected items to have 3 elements, got %d", len(itemsSlice))
	}

	if itemsSlice[0] != int64(1) {
		t.Errorf("Expected first item to be 1, got %v", itemsSlice[0])
	}

	if itemsSlice[1] != int64(2) {
		t.Errorf("Expected second item to be 2, got %v", itemsSlice[1])
	}

	if itemsSlice[2] != int64(3) {
		t.Errorf("Expected third item to be 3, got %v", itemsSlice[2])
	}
}

func TestStreamJSONParserGetNestedObjects(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test getting nested objects
	parser.Append(`{"response":{"user":{"name":"Bob","settings":{"theme":"dark"}}}}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	// Get the entire response object
	response := parser.Get("response")
	responseMap, ok := response.(map[string]interface{})
	if !ok {
		t.Errorf("Expected response to be map[string]interface{}, got %T", response)
	}

	// Get user from response
	user, ok := responseMap["user"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected user to be map[string]interface{}, got %T", responseMap["user"])
	}

	// Verify nested data
	if user["name"] != "Bob" {
		t.Errorf("Expected name to be 'Bob', got %v", user["name"])
	}

	settings, ok := user["settings"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected settings to be map[string]interface{}, got %T", user["settings"])
	}

	if settings["theme"] != "dark" {
		t.Errorf("Expected theme to be 'dark', got %v", settings["theme"])
	}
}

func TestStreamJSONParserGetArrayOfObjects(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test getting array of objects
	parser.Append(`{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	// Get the entire users array
	users := parser.Get("users")
	usersSlice, ok := users.([]interface{})
	if !ok {
		t.Errorf("Expected users to be []interface{}, got %T", users)
	}

	if len(usersSlice) != 2 {
		t.Errorf("Expected users to have 2 elements, got %d", len(usersSlice))
	}

	// Verify first user object
	user1, ok := usersSlice[0].(map[string]interface{})
	if !ok {
		t.Errorf("Expected first user to be map[string]interface{}, got %T", usersSlice[0])
	}

	if user1["id"] != int64(1) {
		t.Errorf("Expected first user id to be 1, got %v", user1["id"])
	}

	if user1["name"] != "Alice" {
		t.Errorf("Expected first user name to be 'Alice', got %v", user1["name"])
	}

	// Verify second user object
	user2, ok := usersSlice[1].(map[string]interface{})
	if !ok {
		t.Errorf("Expected second user to be map[string]interface{}, got %T", usersSlice[1])
	}

	if user2["id"] != int64(2) {
		t.Errorf("Expected second user id to be 2, got %v", user2["id"])
	}

	if user2["name"] != "Bob" {
		t.Errorf("Expected second user name to be 'Bob', got %v", user2["name"])
	}
}

func TestStreamJSONParserGetEmptyObjectAndArray(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test getting empty object and array
	parser.Append(`{"emptyObj":{},"emptyArr":[],"value":42}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	// Get empty object - should return empty map
	emptyObj := parser.Get("emptyObj")
	emptyObjMap, ok := emptyObj.(map[string]interface{})
	if !ok {
		t.Errorf("Expected emptyObj to be map[string]interface{}, got %T", emptyObj)
	}

	if len(emptyObjMap) != 0 {
		t.Errorf("Expected emptyObj to be empty map, got %d elements", len(emptyObjMap))
	}

	// Get empty array - should return empty slice
	emptyArr := parser.Get("emptyArr")
	emptyArrSlice, ok := emptyArr.([]interface{})
	if !ok {
		t.Errorf("Expected emptyArr to be []interface{}, got %T", emptyArr)
	}

	if len(emptyArrSlice) != 0 {
		t.Errorf("Expected emptyArr to be empty slice, got %d elements", len(emptyArrSlice))
	}

	// Value should still work as before
	value := parser.Get("value")
	if value != int64(42) {
		t.Errorf("Expected value to be 42, got %v", value)
	}
}

func TestStreamJSONParserGetRootObject(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test getting the root object
	parser.Append(`{"key1":"value1","key2":123,"key3":true}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	// Get root by calling Get with no keys
	root := parser.Get()
	if root != nil {
		t.Errorf("Expected Get() with no keys to return nil, got %v", root)
	}
}

func TestStreamJSONParserGetRootArray(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test getting the root array
	parser.Append(`[{"id":1},{"id":2},{"id":3}]`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	// Get first element
	first := parser.Get("0")
	firstMap, ok := first.(map[string]interface{})
	if !ok {
		t.Errorf("Expected first element to be map[string]interface{}, got %T", first)
	}

	if firstMap["id"] != int64(1) {
		t.Errorf("Expected first id to be 1, got %v", firstMap["id"])
	}
}

func TestStreamJSONParserGetMixedTypes(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test complex structure with mixed types
	parser.Append(`{
		"string":"text",
		"number":42,
		"bool":true,
		"null":null,
		"object":{"nested":"value"},
		"array":[1,2,3]
	}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	// Test string value
	str := parser.Get("string")
	if str != "text" {
		t.Errorf("Expected string to be 'text', got %v", str)
	}

	// Test number value
	num := parser.Get("number")
	if num != int64(42) {
		t.Errorf("Expected number to be 42, got %v", num)
	}

	// Test boolean value
	boolVal := parser.Get("bool")
	if boolVal != true {
		t.Errorf("Expected bool to be true, got %v", boolVal)
	}

	// Test null value
	nullVal := parser.Get("null")
	if nullVal != nil {
		t.Errorf("Expected null to be nil, got %v", nullVal)
	}

	// Test object - should return map
	obj := parser.Get("object")
	objMap, ok := obj.(map[string]interface{})
	if !ok {
		t.Errorf("Expected object to be map[string]interface{}, got %T", obj)
	}
	if objMap["nested"] != "value" {
		t.Errorf("Expected nested to be 'value', got %v", objMap["nested"])
	}

	// Test array - should return slice
	arr := parser.Get("array")
	arrSlice, ok := arr.([]interface{})
	if !ok {
		t.Errorf("Expected array to be []interface{}, got %T", arr)
	}
	if len(arrSlice) != 3 {
		t.Errorf("Expected array to have 3 elements, got %d", len(arrSlice))
	}
}

func TestStreamJSONParserMultiAppendBoolean(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test multi-append for boolean value with partial retrieval checks
	parser.Append(`{"flag":`)

	// Should be nil since boolean hasn't started yet
	flag := parser.Get("flag")
	if flag != nil {
		t.Errorf("Expected flag to be nil before boolean starts, got %v", flag)
	}

	parser.Append(`tr`)
	// Should still be nil since boolean is incomplete
	flag = parser.Get("flag")
	if flag != nil {
		t.Errorf("Expected flag to be nil for incomplete boolean, got %v", flag)
	}

	parser.Append(`ue`)
	parser.Append(`}`)

	// Now boolean should be complete and available
	flag = parser.Get("flag")
	if flag != true {
		t.Errorf("Expected flag to be true, got %v", flag)
	}

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	// Test false boolean with partial retrieval
	parser2 := NewStreamJSONParser()
	parser2.Append(`{"active":`)

	active := parser2.Get("active")
	if active != nil {
		t.Errorf("Expected active to be nil before boolean starts, got %v", active)
	}

	parser2.Append(`fal`)
	active = parser2.Get("active")
	if active != nil {
		t.Errorf("Expected active to be nil for incomplete boolean, got %v", active)
	}

	parser2.Append(`se`)
	parser2.Append(`}`)

	active = parser2.Get("active")
	if active != false {
		t.Errorf("Expected active to be false, got %v", active)
	}

	if !parser2.IsCompleted() {
		t.Errorf("Expected parser2 to be completed")
	}
}

func TestStreamJSONParserMultiAppendNull(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test multi-append for null value with partial retrieval checks
	parser.Append(`{"data":`)

	// Should be nil since null hasn't started yet (but for different reason)
	data := parser.Get("data")
	if data != nil {
		t.Errorf("Expected data to be nil before null starts, got %v", data)
	}

	parser.Append(`nu`)
	// Should still be nil since null is incomplete
	data = parser.Get("data")
	if data != nil {
		t.Errorf("Expected data to be nil for incomplete null, got %v", data)
	}

	parser.Append(`ll`)
	parser.Append(`}`)

	// Now null should be complete and available
	data = parser.Get("data")
	if data != nil {
		t.Errorf("Expected data to be nil, got %v", data)
	}

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}
}

func TestStreamJSONParserComplexMultiAppend(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test complex JSON with multiple multi-append scenarios
	parser.Append(`{"user":{`)
	parser.Append(`"name":"Jo`)
	parser.Append(`hn Doe"`)
	parser.Append(`,"age":`)
	parser.Append(`3`)
	parser.Append(`0`)
	parser.Append(`,"active":`)
	parser.Append(`tr`)
	parser.Append(`ue`)
	parser.Append(`,"data":`)
	parser.Append(`nu`)
	parser.Append(`ll`)
	parser.Append(`}}`)

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}

	name := parser.Get("user", "name")
	if name != "John Doe" {
		t.Errorf("Expected name to be 'John Doe', got %v", name)
	}

	age := parser.Get("user", "age")
	if age != int64(30) {
		t.Errorf("Expected age to be 30, got %v", age)
	}

	active := parser.Get("user", "active")
	if active != true {
		t.Errorf("Expected active to be true, got %v", active)
	}

	data := parser.Get("user", "data")
	if data != nil {
		t.Errorf("Expected data to be nil, got %v", data)
	}
}

func TestStreamJSONParserNestedPartialAccess(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test nested object partial access during streaming
	parser.Append(`{"response":{`)

	// Should be nil since user object is not started yet
	name := parser.Get("response", "user", "name")
	if name != nil {
		t.Errorf("Expected nested name to be nil, got %v", name)
	}

	parser.Append(`"user":{`)

	// Should still be nil since user object exists but no name yet
	name = parser.Get("response", "user", "name")
	if name != nil {
		t.Errorf("Expected nested name to be nil when user object started, got %v", name)
	}

	parser.Append(`"name":"Alice"`)

	// Now name should be available even though object is not closed
	name = parser.Get("response", "user", "name")
	if name != "Alice" {
		t.Errorf("Expected nested name to be 'Alice', got %v", name)
	}

	parser.Append(`,"age":25`)

	// Name should still be available, but age should be nil (number not terminated yet)
	name = parser.Get("response", "user", "name")
	if name != "Alice" {
		t.Errorf("Expected nested name to be 'Alice', got %v", name)
	}

	age := parser.Get("response", "user", "age")
	if age != nil {
		t.Errorf("Expected nested age to be nil (incomplete number), got %v", age)
	}

	parser.Append(`},"status":"success"}}`)

	// All values should still be available after completion
	name = parser.Get("response", "user", "name")
	if name != "Alice" {
		t.Errorf("Expected nested name to be 'Alice', got %v", name)
	}

	age = parser.Get("response", "user", "age")
	if age != int64(25) {
		t.Errorf("Expected nested age to be 25, got %v", age)
	}

	status := parser.Get("response", "status")
	if status != "success" {
		t.Errorf("Expected status to be 'success', got %v", status)
	}

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}
}

func TestStreamJSONParserArrayPartialAccess(t *testing.T) {
	parser := NewStreamJSONParser()

	// Test partial access to array elements during streaming
	parser.Append(`{"items":[`)

	// Should be nil since no array elements yet
	item0 := parser.Get("items", "0")
	if item0 != nil {
		t.Errorf("Expected first item to be nil, got %v", item0)
	}

	parser.Append(`{"id":1,"name":"Item1"}`)

	// First item should be available
	itemName := parser.Get("items", "0", "name")
	if itemName != "Item1" {
		t.Errorf("Expected first item name to be 'Item1', got %v", itemName)
	}

	parser.Append(`,{"id":2`)

	// First item should still be available, but second item id should be nil (number not terminated)
	itemName = parser.Get("items", "0", "name")
	if itemName != "Item1" {
		t.Errorf("Expected first item name to be 'Item1', got %v", itemName)
	}

	itemId := parser.Get("items", "1", "id")
	if itemId != nil {
		t.Errorf("Expected second item id to be nil (incomplete number), got %v", itemId)
	}

	// Second item name should not be available yet
	item2Name := parser.Get("items", "1", "name")
	if item2Name != nil {
		t.Errorf("Expected second item name to be nil, got %v", item2Name)
	}

	parser.Append(`,"name":"Item2"}]}`)

	// Both items should be complete
	itemName = parser.Get("items", "0", "name")
	if itemName != "Item1" {
		t.Errorf("Expected first item name to be 'Item1', got %v", itemName)
	}

	item2Name = parser.Get("items", "1", "name")
	if item2Name != "Item2" {
		t.Errorf("Expected second item name to be 'Item2', got %v", item2Name)
	}

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}
}

func TestStreamJSONParserIncrementalMessageStatus(t *testing.T) {
	parser := NewStreamJSONParser()

	parser.Append(`{"message":"Hello`)
	// Value should be available even for incomplete string
	msg := parser.Get("message") // "Hello"
	if msg != "Hello" {
		t.Errorf("Expected message to be 'Hello' for incomplete string, got %v", msg)
	}

	parser.Append(` World","status":"`)
	// Now message is complete
	msg = parser.Get("message") // "Hello World"
	if msg != "Hello World" {
		t.Errorf("Expected message to be 'Hello World', got %v", msg)
	}

	parser.Append(`success"}`)
	status := parser.Get("status") // "success"
	if status != "success" {
		t.Errorf("Expected status to be 'success', got %v", status)
	}

	// Verify message value is still correct
	msg = parser.Get("message")
	if msg != "Hello World" {
		t.Errorf("Expected message to still be 'Hello World', got %v", msg)
	}

	if !parser.IsCompleted() {
		t.Errorf("Expected parser to be completed")
	}
}
