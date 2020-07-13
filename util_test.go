package main

import (
	"testing"

	"github.com/alecthomas/participle/lexer"
	assert "github.com/stretchr/testify/require"
)

func TestInArray(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		array    []interface{}
		expected bool
	}{
		{
			name:  "true",
			value: "a",
			array: []interface{}{
				"a",
				"b",
			},
			expected: true,
		},
		{
			name:  "false",
			value: "c",
			array: []interface{}{
				"a",
				"b",
			},
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := inArray(test.value, test.array)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestNotInArray(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		array    []interface{}
		expected bool
	}{
		{
			name:  "false",
			value: "a",
			array: []interface{}{
				"a",
				"b",
			},
			expected: false,
		},
		{
			name:  "true",
			value: "c",
			array: []interface{}{
				"a",
				"b",
			},
			expected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := notInArray(test.value, test.array)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestStringCompare(t *testing.T) {
	tests := []struct {
		name         string
		op           string
		left         string
		right        string
		expectResult bool
		expectError  error
	}{
		{
			name:         "equal true",
			op:           "==",
			left:         "abc",
			right:        "abc",
			expectResult: true,
		},
		{
			name:         "equal false",
			op:           "==",
			left:         "abc",
			right:        "abd",
			expectResult: false,
		},
		{
			name:         "not equal true",
			op:           "!=",
			left:         "abc",
			right:        "abd",
			expectResult: true,
		},
		{
			name:         "not equal false",
			op:           "!=",
			left:         "abc",
			right:        "abc",
			expectResult: false,
		},
		{
			name:         "greater true",
			op:           ">",
			left:         "abd",
			right:        "abc",
			expectResult: true,
		},
		{
			name:         "greater false",
			op:           ">",
			left:         "abc",
			right:        "abd",
			expectResult: false,
		},
		{
			name:         "greater or equal true",
			op:           ">=",
			left:         "abd",
			right:        "abc",
			expectResult: true,
		},
		{
			name:         "greater or equal false",
			op:           ">=",
			left:         "abc",
			right:        "abd",
			expectResult: false,
		},
		{
			name:         "less true",
			op:           "<",
			left:         "abc",
			right:        "abd",
			expectResult: true,
		},
		{
			name:         "less false",
			op:           "<",
			left:         "abd",
			right:        "abc",
			expectResult: false,
		},
		{
			name:         "less or equal true",
			op:           "<=",
			left:         "abc",
			right:        "abd",
			expectResult: true,
		},
		{
			name:         "less or equal false",
			op:           "<=",
			left:         "abd",
			right:        "abc",
			expectResult: false,
		},
		{
			name:         "regexp true",
			op:           "=~",
			left:         "abc",
			right:        "^a",
			expectResult: true,
		},
		{
			name:         "regexp false",
			op:           "=~",
			left:         "abc",
			right:        "^b",
			expectResult: false,
		},
		{
			name:         "not regexp true",
			op:           "!~",
			left:         "abc",
			right:        "^b",
			expectResult: true,
		},
		{
			name:         "not regexp false",
			op:           "!~",
			left:         "abc",
			right:        "^a",
			expectResult: false,
		},
		{
			name:        "regexp invalid",
			op:          "=~",
			left:        "abc",
			right:       "*",
			expectError: newLexerError(0, `failed to parse regexp "*" for string match using =~`),
		},
		{
			name:        "unsupported operator",
			op:          "<>",
			left:        "abc",
			right:       "def",
			expectError: newLexerError(0, `unsupported operator <> for string comparison`),
		},
	}
	pos := lexer.Position{Offset: 0, Column: 1, Line: 1}
	assert := assert.New(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := stringCompare(test.op, test.left, test.right, pos)
			if test.expectError != nil {
				assert.Equal(test.expectError, err)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectResult, actual)
			}
		})
	}
}

func TestUintCompare(t *testing.T) {

}

func TestIntCompare(t *testing.T) {

}

func TestUintBinaryOp(t *testing.T) {
	tests := []struct {
		name         string
		op           string
		left         uint64
		right        uint64
		expectResult uint64
		expectError  error
	}{
		{
			name:         "and",
			op:           "&",
			left:         1,
			right:        0,
			expectResult: 0,
		},
		{
			name:         "or",
			op:           "|",
			left:         1,
			right:        0,
			expectResult: 1,
		},
		{
			name:         "xor",
			op:           "^",
			left:         1,
			right:        1,
			expectResult: 0,
		},
		{
			name:        "invalid",
			op:          "~",
			left:        1,
			right:       1,
			expectError: newLexerError(0, "unsupported integer binary operator ~"),
		},
	}
	pos := lexer.Position{Offset: 0, Column: 1, Line: 1}
	assert := assert.New(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := uintBinaryOp(test.op, test.left, test.right, pos)
			if test.expectError != nil {
				assert.Equal(test.expectError, err)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectResult, actual)
			}
		})
	}
}

func TestIntBinaryOp(t *testing.T) {
	tests := []struct {
		name         string
		op           string
		left         int64
		right        int64
		expectResult int64
		expectError  error
	}{
		{
			name:         "and",
			op:           "&",
			left:         1,
			right:        0,
			expectResult: 0,
		},
		{
			name:         "or",
			op:           "|",
			left:         1,
			right:        0,
			expectResult: 1,
		},
		{
			name:         "xor",
			op:           "^",
			left:         1,
			right:        1,
			expectResult: 0,
		},
		{
			name:        "invalid",
			op:          "~",
			left:        1,
			right:       1,
			expectError: newLexerError(0, "unsupported integer binary operator ~"),
		},
	}
	pos := lexer.Position{Offset: 0, Column: 1, Line: 1}
	assert := assert.New(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := intBinaryOp(test.op, test.left, test.right, pos)
			if test.expectError != nil {
				assert.Equal(test.expectError, err)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectResult, actual)
			}
		})
	}
}

func TestStringBinaryOp(t *testing.T) {
	tests := []struct {
		name         string
		op           string
		left         string
		right        string
		expectResult string
		expectError  error
	}{
		{
			name:         "concat",
			op:           "+",
			left:         "abc",
			right:        "def",
			expectResult: "abcdef",
		},
		{
			name:        "invalid operator",
			op:          "-",
			left:        "abc",
			right:       "def",
			expectError: newLexerError(0, "unsupported string binary operator -"),
		},
	}
	pos := lexer.Position{Offset: 0, Column: 1, Line: 1}
	assert := assert.New(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := stringBinaryOp(test.op, test.left, test.right, pos)
			if test.expectError != nil {
				assert.Equal(test.expectError, err)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectResult, actual)
			}
		})
	}
}

func TestCoerceIntegers(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{
			name:     "int",
			value:    int(55),
			expected: int64(55),
		},
		{
			name:     "int16",
			value:    int16(106),
			expected: int64(106),
		},
		{
			name:     "int32",
			value:    int32(-34),
			expected: int64(-34),
		},
		{
			name:     "int64",
			value:    int64(89),
			expected: int64(89),
		},
		{
			name:     "uint",
			value:    uint(35),
			expected: uint64(35),
		},
		{
			name:     "uint16",
			value:    uint16(5),
			expected: uint64(5),
		},
		{
			name:     "uint32",
			value:    uint32(45),
			expected: uint64(45),
		},
		{
			name:     "uint64",
			value:    uint64(300),
			expected: uint64(300),
		},
		{
			name:     "string",
			value:    "abc",
			expected: "abc",
		},
		{
			name:     "array",
			value:    []int{},
			expected: []int{},
		},
	}
	assert := assert.New(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := coerceIntegers(test.value)
			assert.Equal(test.expected, actual)
		})
	}
}
