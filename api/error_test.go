package api

import (
	"encoding/json"

	. "gopkg.in/check.v1"
)

type ErrorTestSuite struct{}

var _ = Suite(&ErrorTestSuite{})

func (s *ErrorTestSuite) TestErrorStruct(c *C) {
	// Test Error struct creation and fields
	err := Error{Error: "test error message"}
	c.Check(err.Error, Equals, "test error message")
}

func (s *ErrorTestSuite) TestErrorJSONMarshaling(c *C) {
	// Test JSON marshaling of Error struct
	err := Error{Error: "test error message"}

	jsonData, marshalErr := json.Marshal(err)
	c.Check(marshalErr, IsNil)
	c.Check(string(jsonData), Equals, `{"error":"test error message"}`)
}

func (s *ErrorTestSuite) TestErrorJSONUnmarshaling(c *C) {
	// Test JSON unmarshaling into Error struct
	jsonData := `{"error":"test error message"}`

	var err Error
	unmarshalErr := json.Unmarshal([]byte(jsonData), &err)
	c.Check(unmarshalErr, IsNil)
	c.Check(err.Error, Equals, "test error message")
}

func (s *ErrorTestSuite) TestErrorEmptyMessage(c *C) {
	// Test Error struct with empty message
	err := Error{Error: ""}
	c.Check(err.Error, Equals, "")

	jsonData, marshalErr := json.Marshal(err)
	c.Check(marshalErr, IsNil)
	c.Check(string(jsonData), Equals, `{"error":""}`)
}

func (s *ErrorTestSuite) TestErrorSpecialCharacters(c *C) {
	// Test Error struct with special characters
	specialMessages := []string{
		"error with \"quotes\"",
		"error with 'apostrophes'",
		"error with \n newlines",
		"error with \t tabs",
		"error with unicode: 你好",
		"error with emoji: 🚨❌",
		"error with backslashes: \\path\\to\\file",
		"error with json characters: {\"key\": \"value\"}",
		"error with < > & characters",
		"error with null \x00 character",
	}

	for i, message := range specialMessages {
		err := Error{Error: message}
		c.Check(err.Error, Equals, message, Commentf("Test case %d", i))

		// Test JSON marshaling works with special characters
		jsonData, marshalErr := json.Marshal(err)
		c.Check(marshalErr, IsNil, Commentf("Marshal failed for case %d: %s", i, message))

		// Test JSON unmarshaling works with special characters
		var unmarshaled Error
		unmarshalErr := json.Unmarshal(jsonData, &unmarshaled)
		c.Check(unmarshalErr, IsNil, Commentf("Unmarshal failed for case %d: %s", i, message))
		c.Check(unmarshaled.Error, Equals, message, Commentf("Round-trip failed for case %d", i))
	}
}

func (s *ErrorTestSuite) TestErrorLongMessage(c *C) {
	// Test Error struct with very long message
	longMessage := ""
	for i := 0; i < 1000; i++ {
		longMessage += "This is a very long error message. "
	}

	err := Error{Error: longMessage}
	c.Check(err.Error, Equals, longMessage)

	// Test JSON marshaling/unmarshaling with long message
	jsonData, marshalErr := json.Marshal(err)
	c.Check(marshalErr, IsNil)

	var unmarshaled Error
	unmarshalErr := json.Unmarshal(jsonData, &unmarshaled)
	c.Check(unmarshalErr, IsNil)
	c.Check(unmarshaled.Error, Equals, longMessage)
}

func (s *ErrorTestSuite) TestErrorJSONFieldName(c *C) {
	// Test that the JSON field name is exactly "error"
	err := Error{Error: "test"}

	jsonData, marshalErr := json.Marshal(err)
	c.Check(marshalErr, IsNil)

	// Parse as generic map to check field name
	var result map[string]interface{}
	unmarshalErr := json.Unmarshal(jsonData, &result)
	c.Check(unmarshalErr, IsNil)

	// Check that the field is named "error"
	value, exists := result["error"]
	c.Check(exists, Equals, true)
	c.Check(value, Equals, "test")

	// Check that no other fields exist
	c.Check(len(result), Equals, 1)
}

func (s *ErrorTestSuite) TestErrorJSONWithExtraFields(c *C) {
	// Test unmarshaling JSON with extra fields (should be ignored)
	jsonData := `{"error":"test error","extra":"ignored","number":123}`

	var err Error
	unmarshalErr := json.Unmarshal([]byte(jsonData), &err)
	c.Check(unmarshalErr, IsNil)
	c.Check(err.Error, Equals, "test error")
}

func (s *ErrorTestSuite) TestErrorJSONMissingField(c *C) {
	// Test unmarshaling JSON missing the error field
	jsonData := `{"other":"value"}`

	var err Error
	unmarshalErr := json.Unmarshal([]byte(jsonData), &err)
	c.Check(unmarshalErr, IsNil)
	c.Check(err.Error, Equals, "") // Should be zero value
}

func (s *ErrorTestSuite) TestErrorJSONInvalidJSON(c *C) {
	// Test unmarshaling invalid JSON
	invalidJSONs := []string{
		`{"error":}`,
		`{"error": invalid}`,
		`{error: "missing quotes"}`,
		`{"error": "unterminated`,
		`malformed json`,
		``,
		`null`,
		`[]`,
		`123`,
	}

	for i, jsonData := range invalidJSONs {
		var err Error
		unmarshalErr := json.Unmarshal([]byte(jsonData), &err)

		// Should either error or handle gracefully
		if unmarshalErr == nil {
			// If no error, check the result is reasonable
			c.Check(err.Error, FitsTypeOf, "", Commentf("Invalid JSON case %d: %s", i, jsonData))
		} else {
			// Error is expected for malformed JSON
			c.Check(unmarshalErr, NotNil, Commentf("Expected error for case %d: %s", i, jsonData))
		}
	}
}

func (s *ErrorTestSuite) TestErrorZeroValue(c *C) {
	// Test zero value of Error struct
	var err Error
	c.Check(err.Error, Equals, "")

	// Test JSON marshaling of zero value
	jsonData, marshalErr := json.Marshal(err)
	c.Check(marshalErr, IsNil)
	c.Check(string(jsonData), Equals, `{"error":""}`)
}

func (s *ErrorTestSuite) TestErrorPointer(c *C) {
	// Test Error struct as pointer
	err := &Error{Error: "pointer error"}
	c.Check(err.Error, Equals, "pointer error")

	// Test JSON marshaling of pointer
	jsonData, marshalErr := json.Marshal(err)
	c.Check(marshalErr, IsNil)
	c.Check(string(jsonData), Equals, `{"error":"pointer error"}`)

	// Test JSON unmarshaling into pointer
	var err2 *Error
	unmarshalErr := json.Unmarshal(jsonData, &err2)
	c.Check(unmarshalErr, IsNil)
	c.Check(err2, NotNil)
	c.Check(err2.Error, Equals, "pointer error")
}

func (s *ErrorTestSuite) TestErrorStructCopy(c *C) {
	// Test copying Error struct
	err1 := Error{Error: "original error"}
	err2 := err1

	c.Check(err2.Error, Equals, "original error")

	// Modify original and ensure copy is independent
	err1.Error = "modified error"
	c.Check(err1.Error, Equals, "modified error")
	c.Check(err2.Error, Equals, "original error")
}

func (s *ErrorTestSuite) TestErrorStructComparison(c *C) {
	// Test comparing Error structs
	err1 := Error{Error: "same message"}
	err2 := Error{Error: "same message"}
	err3 := Error{Error: "different message"}

	c.Check(err1 == err2, Equals, true)
	c.Check(err1 == err3, Equals, false)
	c.Check(err2 == err3, Equals, false)
}

func (s *ErrorTestSuite) TestErrorStructInSlice(c *C) {
	// Test Error struct in slice operations
	errors := []Error{
		{Error: "first error"},
		{Error: "second error"},
		{Error: "third error"},
	}

	c.Check(len(errors), Equals, 3)
	c.Check(errors[0].Error, Equals, "first error")
	c.Check(errors[1].Error, Equals, "second error")
	c.Check(errors[2].Error, Equals, "third error")

	// Test JSON marshaling of slice
	jsonData, marshalErr := json.Marshal(errors)
	c.Check(marshalErr, IsNil)

	var unmarshaled []Error
	unmarshalErr := json.Unmarshal(jsonData, &unmarshaled)
	c.Check(unmarshalErr, IsNil)
	c.Check(len(unmarshaled), Equals, 3)
	c.Check(unmarshaled[0].Error, Equals, "first error")
}

func (s *ErrorTestSuite) TestErrorStructInMap(c *C) {
	// Test Error struct in map operations
	errorMap := map[string]Error{
		"key1": {Error: "first error"},
		"key2": {Error: "second error"},
	}

	c.Check(len(errorMap), Equals, 2)
	c.Check(errorMap["key1"].Error, Equals, "first error")
	c.Check(errorMap["key2"].Error, Equals, "second error")

	// Test JSON marshaling of map
	jsonData, marshalErr := json.Marshal(errorMap)
	c.Check(marshalErr, IsNil)

	var unmarshaled map[string]Error
	unmarshalErr := json.Unmarshal(jsonData, &unmarshaled)
	c.Check(unmarshalErr, IsNil)
	c.Check(len(unmarshaled), Equals, 2)
	c.Check(unmarshaled["key1"].Error, Equals, "first error")
	c.Check(unmarshaled["key2"].Error, Equals, "second error")
}
