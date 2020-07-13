package main

import (
	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

var (
	expressionLexer = lexer.Must(ebnf.New(`
		Hex = ("0" "x") hexdigit { hexdigit } .
		Ident = (alpha | "_") { "_" | "." | alpha | digit } .
		String = "\"" { "\u0000"…"\uffff"-"\""-"\\" | "\\" any } "\"" .
		UnixSystemPath = "/" alpha { alpha | digit | "-" | "." | "_" | "/" } ["*" [ "." { alpha | digit } ] ].
		Octal = "0" octaldigit { octaldigit } .
		Decimal = [ "-" | "+" ] digit { digit } .
		Punct = "!"…"/" | ":"…"@" | "["…` + "\"`\"" + ` | "{"…"~" .
		Whitespace = ( " " | "\t" ) { " " | "\t" } .
		alpha = "a"…"z" | "A"…"Z" .
		octaldigit = "0"…"7" .
		hexdigit = "A"…"F" | "a"…"f" | digit .
		digit = "0"…"9" .
		any = "\u0000"…"\uffff" .
	`))

	expressionOptions = []participle.Option{
		participle.Lexer(expressionLexer),
		participle.Unquote("String"),
		participle.UseLookahead(2),
		participle.Elide("Whitespace"),
	}

	expressionParser = participle.MustBuild(&Expression{}, expressionOptions...)

	iterableParser = participle.MustBuild(&IterableExpression{}, expressionOptions...)

	pathParser = participle.MustBuild(&PathExpression{}, expressionOptions...)
)

// ParseExpression parses Expression from a string
func ParseExpression(s string) (*Expression, error) {
	expr := &Expression{}
	err := expressionParser.ParseString(s, expr)
	if err != nil {
		return nil, err
	}
	return expr, nil
}

// ParseIterable parses IterableExpression from a string
func ParseIterable(s string) (*IterableExpression, error) {
	expr := &IterableExpression{}
	err := iterableParser.ParseString(s, expr)
	if err != nil {
		return nil, err
	}
	return expr, nil
}

// ParsePath parses PathExpression from a string
func ParsePath(s string) (*PathExpression, error) {
	expr := &PathExpression{}
	err := pathParser.ParseString(s, expr)
	if err != nil {
		return nil, err
	}
	return expr, nil
}
