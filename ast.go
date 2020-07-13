package main

import (
	"github.com/alecthomas/participle/lexer"
)

// Expression represents basic expression syntax that can be evaluated for an Instance
type Expression struct {
	Pos lexer.Position

	Comparison *Comparison `@@`
	Op         *string     `[ @( "|" "|" | "&" "&" )`
	Next       *Expression `  @@ ]`
}

// Iterable represents an iterable expration that can be evaluated for an Iterator
type IterableExpression struct {
	Pos lexer.Position

	IterableComparison *IterableComparison `@@`
	Expression         *Expression         `| @@`
}

// IterableComparison allows evaluating a builtin pseudo-funciion for an iterable expression
type IterableComparison struct {
	Pos lexer.Position

	Fn               *string           `@Ident`
	Expression       *Expression       `"(" @@ ")"`
	ScalarComparison *ScalarComparison `[ @@ ]`
}

// PathExpression represents an expression evaluating to a file path or file glob
type PathExpression struct {
	Pos lexer.Position

	Path       *string     `@UnixSystemPath`
	Expression *Expression `| @@`
}

// Comparison represents syntax for comparison operations
type Comparison struct {
	Pos lexer.Position

	Term             *Term             `@@`
	ScalarComparison *ScalarComparison `[ @@`
	ArrayComparison  *ArrayComparison  `| @@ ]`
}

// ScalarComparison represents syntax for scalar comparison
type ScalarComparison struct {
	Pos lexer.Position

	Op   *string     `@( ">" "=" | "<" "=" | ">" | "<" | "!" "=" | "=" "=" | "=" "~" | "!" "~" )`
	Next *Comparison `  @@`
}

// ArrayComparison represents syntax for array comparison
type ArrayComparison struct {
	Pos lexer.Position

	Op *string ` ( @( "in" | "not" "in" )`
	// TODO: FIXME: likely doesn't work with rhs expression
	Array *Array `@@ )`
}

// Term is an abstract term allowing optional binary bit operation syntax
type Term struct {
	Pos lexer.Position

	Unary *Unary  `@@`
	Op    *string `[ @( "&" | "|" | "^" | "+" )`
	Next  *Term   `  @@ ]`
}

// Unary is a unary bit operation syntax
type Unary struct {
	Pos lexer.Position

	Op    *string `  ( @( "!" | "-" | "^" )`
	Unary *Unary  `    @@ )`
	Value *Value  `| @@`
}

// Array provides support for array syntax and may contain any valid Values (mixed allowed)
type Array struct {
	Pos lexer.Position

	Values []Value `"[" @@ { "," @@ } "]"`

	Ident *string `| @Ident`
}

// Value provides support for various value types in expression including
// integers in various form, strings, function calls, variables and
// subexpressions
type Value struct {
	Pos lexer.Position

	Hex           *string     `  @Hex`
	Octal         *string     `| @Octal`
	Decimal       *int64      `| @Decimal`
	String        *string     `| @String`
	Call          *Call       `| @@`
	Variable      *string     `| @Ident`
	Subexpression *Expression `| "(" @@ ")"`
}

// Call implements function call syntax
type Call struct {
	Pos lexer.Position

	Name string        `@Ident`
	Args []*Expression `"(" [ @@ { "," @@ } ] ")"`
}
