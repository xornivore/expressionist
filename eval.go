package main

import (
	"reflect"
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

type Iterator interface {
	Next() (*Context, error)
	Done() bool
}

func (e *IterableExpression) Evaluate(it Iterator) (bool, error) {
	if e.IterableComparison == nil {
		result := true
		err := e.iterate(it, e.Expression, func(ctx *Context, passed bool) bool {
			if !passed {
				result = false
				// first failure terminates the iteration
				// TODO: add reporting of ctx value here
				return false
			}
			return true
		})

		if err != nil {
			return false, err
		}
		return result, nil
	}

	if e.IterableComparison.Fn == nil {
		return false, lexer.Errorf(e.Pos, "expecting function for iterable comparison")
	}

	totalCount := 0
	passedCount := 0
	err := e.iterate(it, e.IterableComparison.Expression, func(ctx *Context, passed bool) bool {
		totalCount++
		if passed {
			passedCount++
		}
		// TODO: can evaluate some pseudo-function cases here without the need to go through the entire list
		return true
	})
	if err != nil {
		return false, err
	}

	switch *e.IterableComparison.Fn {
	case "all":
		return passedCount == totalCount, nil

	case "none":
		return passedCount == 0, nil

	case "any":
		return passedCount != 0, nil

	case "len":
		if e.IterableComparison.ScalarComparison == nil {
			return false, lexer.Errorf(e.Pos, "expecting rhs of len() expression")
		}

		if e.IterableComparison.ScalarComparison.Op == nil {
			return false, lexer.Errorf(e.Pos, "expecting operator for len() comparison")
		}

		rhs, err := e.IterableComparison.ScalarComparison.Next.Evaluate(&Context{}) // TODO: need some global context here
		if err != nil {
			return false, err
		}

		expectedCount, ok := rhs.(int64)
		if !ok {
			return false, lexer.Errorf(e.Pos, "expecting rhs for len() iterable comparison to be integer")
		}

		return compareInts(*e.IterableComparison.ScalarComparison.Op, int64(passedCount), expectedCount, e.Pos)
	default:
		return false, lexer.Errorf(e.Pos, "unexpected function for iterable comparison %s", *e.IterableComparison.Fn)
	}
}

func (e *IterableExpression) iterate(it Iterator, expression *Expression, check func(ctx *Context, passed bool) bool) error {
	for !it.Done() {
		ctx, err := it.Next()
		if err != nil {
			return err
		}

		v, err := expression.Evaluate(ctx)
		if err != nil {
			return err
		}
		passed, ok := v.(bool)
		if !ok {
			return lexer.Errorf(e.Pos, "expected a boolean resuls of evaluation")
		}

		if !check(ctx, passed) {
			break
		}
	}
	return nil
}

func (e *PathExpression) Evaluate(ctx *Context) (interface{}, error) {
	if e.Path != nil {
		return *e.Path, nil
	}
	return e.Expression.Evaluate(ctx)
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
		if c.ArrayComparison.Array == nil {
			return nil, lexer.Errorf(c.Pos, "missing rhs of array operation %s", *c.ArrayComparison.Op)
		}

		rhs, err := c.ArrayComparison.Array.Evaluate(ctx)
		if err != nil {
			return nil, err
		}

		array, ok := rhs.([]interface{})
		if !ok {
			return nil, lexer.Errorf(c.Pos, "expecting rhs of array operation %s to be an array", *c.ArrayComparison.Op)
		}

		switch *c.ArrayComparison.Op {
		case "in":
			return c.inArray(lhs, array)

		case "not in":
			return c.notInArray(lhs, array)
		default:
			return nil, lexer.Errorf(c.Pos, "unsupported array operation %s", *c.ArrayComparison.Op)
		}

	case c.ScalarComparison != nil:
		if c.ScalarComparison.Next == nil {
			return nil, lexer.Errorf(c.Pos, "missing rhs of %s", *c.ScalarComparison.Op)
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

func (c *Comparison) inArray(value interface{}, array []interface{}) (interface{}, error) {
	for _, rhs := range array {
		if reflect.DeepEqual(value, rhs) {
			return true, nil
		}
	}
	return false, nil
}

func (c *Comparison) notInArray(value interface{}, array []interface{}) (interface{}, error) {
	for _, rhs := range array {
		if reflect.DeepEqual(value, rhs) {
			return true, nil
		}
	}
	return false, nil
}

func (c *Comparison) compare(lhs, rhs interface{}, op string) (interface{}, error) {
	switch lhs := lhs.(type) {
	case int64:

		rhs, ok := rhs.(int64)
		if !ok {
			return nil, lexer.Errorf(c.Pos, "rhs of %s must be an integer", op)
		}
		return compareInts(op, lhs, rhs, c.Pos)
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

func compareInts(op string, lhs, rhs int64, pos lexer.Position) (bool, error) {
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
		return false, lexer.Errorf(pos, "unsupported operator %s for integer comparison", op)
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

func (a *Array) Evaluate(ctx *Context) (interface{}, error) {
	if a.Ident != nil {
		value, ok := ctx.Vars[*a.Ident]
		if !ok {
			return nil, lexer.Errorf(a.Pos, "unknown variable %q used as array", *a.Ident)
		}
		return value, nil
	}
	var result []interface{}
	for _, value := range a.Values {
		v, err := value.Evaluate(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
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
