package main

import (
	"errors"
	"strconv"

	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/repr"
)

type Evaluatable interface {
	Evaluate(ctx *Context) (interface{}, error)
}

type Function func(args ...interface{}) (interface{}, error)

// Context for evaluation.
type Context struct {
	// User-provided functions.
	Functions map[string]Function
	// Vars defined during evaluation.
	Vars map[string]interface{}
}

func (e *Expression) Evaluate(ctx *Context) (interface{}, error) {
	lhs, err := e.Comparison.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	if e.Next == nil {
		return lhs, nil
	}

	left, ok := lhs.(bool)
	if !ok {
		return nil, lexer.Errorf(e.Pos, "type mismatch, expected bool in lhs of boolean expression")
	}

	rhs, err := e.Next.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	right := rhs.(bool)

	switch *e.Op {
	case "&&":
		return left && right, nil
	case "||":
		return left || right, nil
	}
	return nil, lexer.Errorf(e.Pos, "unsupported operator %s in boolean expression", *e.Op)
}

func (c *Comparison) Evaluate(ctx *Context) (interface{}, error) {
	lhs, err := c.Term.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	switch {
	case c.ArrayComparison != nil:
		return nil, errors.New("array comparison not yet supported")
	case c.ScalarComparison != nil:
		if c.ScalarComparison.Next == nil {
			return nil, errors.New("missing right hand side of comparison")
		}
		rhs, err := c.ScalarComparison.Next.Evaluate(ctx)
		if err != nil {
			return nil, err
		}
		return c.compare(lhs, rhs, *c.ScalarComparison.Op)

	default:
		return lhs, nil
	}
}

func (c *Comparison) compare(lhs, rhs interface{}, op string) (interface{}, error) {
	switch lhs := lhs.(type) {
	case int64:
		rhs, ok := rhs.(int64)
		if !ok {
			return nil, lexer.Errorf(c.Pos, "rhs of %s must be an integer", op)
		}
		switch op {
		case "==":
			return lhs == rhs, nil
		case "!=":
			return lhs != rhs, nil
		case "<":
			return lhs < rhs, nil
		case ">":
			return lhs > rhs, nil
		case "<=":
			return lhs <= rhs, nil
		case ">=":
			return lhs >= rhs, nil

		default:
			return nil, lexer.Errorf(c.Pos, "unsupported operator %s for integer comparison", op)
		}
	case string:
		rhs, ok := rhs.(string)
		if !ok {
			return nil, lexer.Errorf(c.Pos, "rhs of %s must be a string", op)
		}
		switch op {
		case "==":
			return lhs == rhs, nil
		case "!=":
			return lhs != rhs, nil
		case "<":
			return lhs < rhs, nil
		case ">":
			return lhs > rhs, nil
		case "<=":
			return lhs <= rhs, nil
		case ">=":
			return lhs >= rhs, nil
		default:
			return nil, lexer.Errorf(c.Pos, "unsupported operator %s for string comparison", op)
		}
	default:
		return nil, lexer.Errorf(c.Pos, "lhs of %s must be a number or string", op)
	}
}

func (t *Term) Evaluate(ctx *Context) (interface{}, error) {
	lhs, err := t.Unary.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	if t.Op == nil {
		return lhs, nil
	}

	if t.Next == nil {
		return nil, lexer.Errorf(t.Pos, "expected rhs in binary bit operation")
	}

	rhs, err := t.Next.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	switch lhs := lhs.(type) {
	case int64:
		rhs, ok := rhs.(int64)
		if !ok {
			return nil, lexer.Errorf(t.Pos, "rhs of %s must be an integer", *t.Op)
		}

		switch *t.Op {
		case "&":
			return lhs & rhs, nil
		case "|":
			return lhs | rhs, nil
		case "^":
			return lhs ^ rhs, nil
		default:
			return nil, lexer.Errorf(t.Pos, "unsupported binary operator %s", *t.Op)
		}

	default:
		return nil, lexer.Errorf(t.Pos, "binary bit operation not supported for this type")
	}
}

func (u *Unary) Evaluate(ctx *Context) (interface{}, error) {
	if u.Value != nil {
		return u.Value.Evaluate(ctx)
	}

	if u.Unary == nil || u.Op == nil {
		return nil, lexer.Errorf(u.Pos, "invalid unary operation")
	}

	rhs, err := u.Unary.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	switch *u.Op {
	case "!":
		rhs, ok := rhs.(bool)
		if !ok {
			return nil, lexer.Errorf(u.Pos, "rhs of %s must be a boolean", *u.Op)
		}
		return !rhs, nil
	case "-":
		rhs, ok := rhs.(int64)
		if !ok {
			return nil, lexer.Errorf(u.Pos, "rhs of %s must be an integer", *u.Op)
		}
		return -rhs, nil
	case "^":
		rhs, ok := rhs.(int64)
		if !ok {
			return nil, lexer.Errorf(u.Pos, "rhs of %s must be an integer", *u.Op)
		}
		return ^rhs, nil
	default:
		return nil, lexer.Errorf(u.Pos, "unsupported unary operator %s", *u.Op)
	}
}

func (v *Value) Evaluate(ctx *Context) (interface{}, error) {
	switch {
	case v.Hex != nil:
		return strconv.ParseInt(*v.Hex, 0, 64)
	case v.Octal != nil:
		return strconv.ParseInt(*v.Octal, 8, 64)
	case v.Decimal != nil:
		return *v.Decimal, nil
	case v.String != nil:
		return *v.String, nil
	case v.Variable != nil:
		value, ok := ctx.Vars[*v.Variable]
		if !ok {
			return nil, lexer.Errorf(v.Pos, "unknown variable %q", *v.Variable)
		}
		return value, nil
	case v.Subexpression != nil:
		return v.Subexpression.Evaluate(ctx)
	case v.Call != nil:
		return v.Call.Evaluate(ctx)
	}

	return nil, lexer.Errorf(v.Pos, "unsupported value type %s", repr.String(v))
}

func (c *Call) Evaluate(ctx *Context) (interface{}, error) {
	function, ok := ctx.Functions[c.Name]
	if !ok {
		return nil, lexer.Errorf(c.Pos, "unknown function %q", c.Name)
	}
	args := []interface{}{}
	for _, arg := range c.Args {
		value, err := arg.Evaluate(ctx)
		if err != nil {
			return nil, err
		}
		args = append(args, value)
	}

	value, err := function(args...)
	if err != nil {
		return nil, lexer.Errorf(c.Pos, "call to %s() failed", c.Name)
	}
	return value, nil
}

/*
func (f *Factor) Evaluate(ctx *Context) (interface{}, error) {
	base, err := f.Base.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if f.Exponent == nil {
		return base, nil
	}
	baseNum, exponentNum, err := evaluateFloats(ctx, base, f.Exponent)
	if err != nil {
		return nil, lexer.Errorf(f.Pos, "invalid factor: %s", err)
	}
	return math.Pow(baseNum, exponentNum), nil
}

func (o *OpFactor) Evaluate(ctx *Context, lhs interface{}) (interface{}, error) {
	lhsNumber, rhsNumber, err := evaluateFloats(ctx, lhs, o.Factor)
	if err != nil {
		return nil, lexer.Errorf(o.Pos, "invalid arguments for %s: %s", o.Operator, err)
	}
	switch o.Operator {
	case "*":
		return lhsNumber * rhsNumber, nil
	case "/":
		return lhsNumber / rhsNumber, nil
	}
	panic("unreachable")
}

func (t *Term) Evaluate(ctx *Context) (interface{}, error) {
	lhs, err := t.Left.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	for _, right := range t.Right {
		rhs, err := right.Evaluate(ctx, lhs)
		if err != nil {
			return nil, err
		}
		lhs = rhs
	}
	return lhs, nil
}

func (o *OpTerm) Evaluate(ctx *Context, lhs interface{}) (interface{}, error) {
	lhsNumber, rhsNumber, err := evaluateFloats(ctx, lhs, o.Term)
	if err != nil {
		return nil, lexer.Errorf(o.Pos, "invalid arguments for %s: %s", o.Operator, err)
	}
	switch o.Operator {
	case "+":
		return lhsNumber + rhsNumber, nil
	case "-":
		return lhsNumber - rhsNumber, nil
	}
	panic("unreachable")
}

func (c *Cmp) Evaluate(ctx *Context) (interface{}, error) {
	lhs, err := c.Left.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	for _, right := range c.Right {
		rhs, err := right.Evaluate(ctx, lhs)
		if err != nil {
			return nil, err
		}
		lhs = rhs
	}
	return lhs, nil
}

func (o *OpCmp) Evaluate(ctx *Context, lhs interface{}) (interface{}, error) {
	rhs, err := o.Cmp.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	switch lhs := lhs.(type) {
	case float64:
		rhs, ok := rhs.(float64)
		if !ok {
			return nil, lexer.Errorf(o.Pos, "rhs of %s must be a number", o.Operator)
		}
		switch o.Operator {
		case "==":
			return lhs == rhs, nil
		case "!=":
			return lhs != rhs, nil
		case "<":
			return lhs < rhs, nil
		case ">":
			return lhs > rhs, nil
		case "<=":
			return lhs <= rhs, nil
		case ">=":
			return lhs >= rhs, nil
		}
	case string:
		rhs, ok := rhs.(string)
		if !ok {
			return nil, lexer.Errorf(o.Pos, "rhs of %s must be a string", o.Operator)
		}
		switch o.Operator {
		case "==":
			return lhs == rhs, nil
		case "!=":
			return lhs != rhs, nil
		case "<":
			return lhs < rhs, nil
		case ">":
			return lhs > rhs, nil
		case "<=":
			return lhs <= rhs, nil
		case ">=":
			return lhs >= rhs, nil
		}
	default:
		return nil, lexer.Errorf(o.Pos, "lhs of %s must be a number or string", o.Operator)
	}
	panic("unreachable")
}

func (e *Expression) Evaluate(ctx *Context) (interface{}, error) {
	lhs, err := e.Left.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	for _, right := range e.Right {
		rhs, err := right.Evaluate(ctx, lhs)
		if err != nil {
			return nil, err
		}
		lhs = rhs
	}
	return lhs, nil
}



func evaluateFloats(ctx *Context, lhs interface{}, rhsExpr Evaluatable) (float64, float64, error) {
	rhs, err := rhsExpr.Evaluate(ctx)
	if err != nil {
		return 0, 0, err
	}
	lhsNumber, ok := lhs.(float64)
	if !ok {
		return 0, 0, fmt.Errorf("lhs must be a number")
	}
	rhsNumber, ok := rhs.(float64)
	if !ok {
		return 0, 0, fmt.Errorf("rhs must be a number")
	}
	return lhsNumber, rhsNumber, nil
}
*/
