package main

import (
	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
)

var (
	expressionLexer = lexer.Must(ebnf.New(`
		Hex = ("0" "x") { hexdigit } .
		Ident = (alpha | "_") { "_" | "." | alpha | digit } .
		String = "\"" { "\u0000"…"\uffff"-"\""-"\\" | "\\" any } "\"" .
		Octal = "0" { octaldigit } .
		Decimal = [ "-" | "+" ] digit { digit } .
		Punct = "!"…"/" | ":"…"@" | "["…` + "\"`\"" + ` | "{"…"~" .
		Whitespace = ( " " | "\t" ) { " " | "\t" } .
		alpha = "a"…"z" | "A"…"Z" .
		octaldigit = "0"…"7" .
		hexdigit = "A"…"F" | "a"…"f" | digit .
		digit = "0"…"9" .
		any = "\u0000"…"\uffff" .
	`))

	expressionParser = participle.MustBuild(&Expression{},
		participle.Lexer(expressionLexer),
		participle.Unquote("String"),
		participle.UseLookahead(2),
		participle.Elide("Whitespace"),
	)

	iterableParser = participle.MustBuild(&IterableExpression{},
		participle.Lexer(expressionLexer),
		participle.Unquote("String"),
		participle.UseLookahead(2),
		participle.Elide("Whitespace"),
	)
)
