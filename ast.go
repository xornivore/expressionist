package main

import (
	"github.com/alecthomas/participle/lexer"
)

func ParseExpression(s string) (*Expression, error) {
	expr := &Expression{}
	err := expressionParser.ParseString(s, expr)
	if err != nil {
		return nil, err
	}
	return expr, nil
}

func ParseIterable(s string) (*IterableExpression, error) {
	expr := &IterableExpression{}
	err := iterableParser.ParseString(s, expr)
	if err != nil {
		return nil, err
	}
	return expr, nil
}

type Expression struct {
	Pos lexer.Position

	Comparison *Comparison `@@`
	Op         *string     `[ @( "|" "|" | "&" "&" )`
	Next       *Expression `  @@ ]`
}

type IterableExpression struct {
	Pos lexer.Position

	IterableComparison *IterableComparison `@@`
	Expression         *Expression         `| @@`
}

type IterableComparison struct {
	Pos lexer.Position

	Fn               *string           `@Ident`
	Expression       *Expression       `"(" @@ ")"`
	ScalarComparison *ScalarComparison `[ @@ ]`
}

type Comparison struct {
	Pos lexer.Position

	Term             *Term             `@@`
	ScalarComparison *ScalarComparison `[ @@`
	ArrayComparison  *ArrayComparison  `| @@ ]`
}

type ScalarComparison struct {
	Pos lexer.Position

	Op   *string     `@( ">" | ">" "=" | "<" | "<" "=" | "!" "=" | "=" "=" | "=" "~" | "!" "~" )`
	Next *Comparison `  @@`
}

type ArrayComparison struct {
	Pos lexer.Position

	Op    *string ` ( @( "in" | "not" "in" )`
	Array *Array  `@@ )`
}

type Term struct {
	Pos lexer.Position

	Unary *Unary  `@@`
	Op    *string `[ @( "&" | "|" | "^" )`
	Next  *Term   `  @@ ]`
}

type Unary struct {
	Pos lexer.Position

	Op    *string `  ( @( "!" | "-" | "^" )`
	Unary *Unary  `    @@ )`
	Value *Value  `| @@`
}

type Array struct {
	Pos lexer.Position

	Values []Value `"[" @@ { "," @@ } "]"`

	Ident *string `| @Ident`
}

type Value struct {
	Pos lexer.Position

	Hex           *string     `  @Hex`
	Octal         *string     `| @Octal`
	Decimal       *int64      `| @Int`
	String        *string     `| @String`
	Call          *Call       `| @@`
	Variable      *string     `| @Ident`
	Subexpression *Expression `| "(" @@ ")"`
}

type Call struct {
	Pos lexer.Position

	Name string        `@Ident`
	Args []*Expression `"(" [ @@ { "," @@ } ] ")"`
}
