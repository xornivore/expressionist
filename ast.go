package main

import (
	"github.com/alecthomas/participle/lexer"
)

func Parse(s string) (*Expression, error) {
	expr := &Expression{}
	err := parser.ParseString(s, expr)
	if err != nil {
		return nil, err
	}
	return expr, nil
}

type Expression struct {
	Pos lexer.Position

	// EnumFunction `[ @( "len" )`
	Comparison *Comparison `@@`
	Op         *string     `[ @( "|" "|" | "&" "&" )`
	Next       *Expression `  @@ ]`
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

	Strings []string `"[" @String { "," @String } "]"`
	Numbers []int    `| "[" @Int { "," @Int } "]"`
	// TODO - lists of octal and hex numbers once needed
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
