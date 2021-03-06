package main

import (
	"strconv"

	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/repr"
)

// Evaluatable abstracts part of an expression that can be evaluated for an instance
type Evaluatable interface {
	Evaluate(instance *Instance) (interface{}, error)
}

// Function describes a function callable for an instance
type Function func(instance *Instance, args ...interface{}) (interface{}, error)

// FunctionMap describes a map of functions
type FunctionMap map[string]Function

// VarMap describes a map of variables
type VarMap map[string]interface{}

// Instance for evaluation
type Instance struct {
	// Instance functions
	Functions FunctionMap
	// Vars defined during evaluation.
	Vars VarMap
}

// Iterator abstracts iteration over a set of instances for expression evaluation
type Iterator interface {
	Next() (*Instance, error)
	Done() bool
}

// InstanceResult captures an Instance along with the passed or failed status for the result
type InstanceResult struct {
	Instance *Instance
	Passed   bool
}

// Evaluate evaluates an iterable expression for an iterator
func (e *IterableExpression) Evaluate(it Iterator, global *Instance) (*InstanceResult, error) {
	if e.IterableComparison == nil {
		return e.iterate(
			it,
			e.Expression,
			func(instance *Instance, passed bool) bool {
				// First failure stops the iteration
				return !passed
			},
		)
	}

	if e.IterableComparison.Fn == nil {
		return nil, lexer.Errorf(e.Pos, "expecting function for iterable comparison")
	}

	fn := *e.IterableComparison.Fn

	totalCount := 0
	passedCount := 0
	result, err := e.iterate(
		it,
		e.IterableComparison.Expression, func(instance *Instance, passed bool) bool {
			totalCount++
			if passed {
				passedCount++
				if fn == "any" || fn == "none" {
					return false
				}
			} else if fn == "all" {
				return false
			}
			return true
		},
	)
	if err != nil {
		return nil, err
	}

	passed, err := e.evaluatePassed(global, passedCount, totalCount)
	if err != nil {
		return nil, err
	}

	return &InstanceResult{
		Instance: result.Instance,
		Passed:   passed,
	}, nil
}

func (e *IterableExpression) evaluatePassed(global *Instance, passedCount, totalCount int) (bool, error) {
	switch *e.IterableComparison.Fn {
	case "all":
		return passedCount == totalCount, nil

	case "none":
		return passedCount == 0, nil

	case "any":
		return passedCount != 0, nil

	case "len":
		if e.IterableComparison.ScalarComparison == nil {
			return false, lexer.Errorf(e.Pos, "expecting rhs of iterable comparison using len()")
		}

		if e.IterableComparison.ScalarComparison.Op == nil {
			return false, lexer.Errorf(e.Pos, "expecting operator for iterable comparison using len()")
		}

		rhs, err := e.IterableComparison.ScalarComparison.Next.Evaluate(global)
		if err != nil {
			return false, err
		}

		expectedCount, ok := rhs.(int64)
		if !ok {
			return false, lexer.Errorf(e.Pos, "expecting an integer rhs for iterable comparison using len()")
		}

		return intCompare(*e.IterableComparison.ScalarComparison.Op, int64(passedCount), expectedCount, e.Pos)
	default:
		return false, lexer.Errorf(e.Pos, `unexpected function "%s()" for iterable comparison`, *e.IterableComparison.Fn)
	}
}

func (e *IterableExpression) iterate(it Iterator, expression *Expression, checkResult func(instance *Instance, passed bool) bool) (*InstanceResult, error) {
	var (
		instance *Instance
		err      error
		passed   bool
	)
	for !it.Done() {
		instance, err = it.Next()
		if err != nil {
			return nil, err
		}

		v, err := expression.Evaluate(instance)
		if err != nil {
			return nil, err
		}

		var ok bool
		if passed, ok = v.(bool); !ok {
			return nil, lexer.Errorf(e.Pos, "expected a boolean resuls of evaluation")
		}

		if !checkResult(instance, passed) {
			break
		}
	}
	return &InstanceResult{
		Instance: instance,
		Passed:   passed,
	}, nil
}

func (e *PathExpression) Evaluate(instance *Instance) (interface{}, error) {
	if e.Path != nil {
		return *e.Path, nil
	}
	return e.Expression.Evaluate(instance)
}

func (e *Expression) Evaluate(instance *Instance) (interface{}, error) {
	lhs, err := e.Comparison.Evaluate(instance)
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

	rhs, err := e.Next.Evaluate(instance)
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

func (c *Comparison) Evaluate(instance *Instance) (interface{}, error) {
	lhs, err := c.Term.Evaluate(instance)
	if err != nil {
		return nil, err
	}
	switch {
	case c.ArrayComparison != nil:
		if c.ArrayComparison.Array == nil {
			return nil, lexer.Errorf(c.Pos, "missing rhs of array operation %s", *c.ArrayComparison.Op)
		}

		rhs, err := c.ArrayComparison.Array.Evaluate(instance)
		if err != nil {
			return nil, err
		}

		array, ok := rhs.([]interface{})
		if !ok {
			return nil, lexer.Errorf(c.Pos, "rhs of %s array operation must be an array", *c.ArrayComparison.Op)
		}

		switch *c.ArrayComparison.Op {
		case "in":
			return inArray(lhs, array), nil
		case "notin":
			return notInArray(lhs, array), nil
		default:
			return nil, lexer.Errorf(c.Pos, "unsupported array operation %s", *c.ArrayComparison.Op)
		}

	case c.ScalarComparison != nil:
		if c.ScalarComparison.Next == nil {
			return nil, lexer.Errorf(c.Pos, "missing rhs of %s", *c.ScalarComparison.Op)
		}
		rhs, err := c.ScalarComparison.Next.Evaluate(instance)
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
	case uint64:
		switch rhs := rhs.(type) {
		case uint64:
			return uintCompare(op, lhs, rhs, c.Pos)
		case int64:
			return uintCompare(op, lhs, uint64(rhs), c.Pos)
		default:
			return nil, lexer.Errorf(c.Pos, "rhs of %s must be an integer", op)
		}
	case int64:
		switch rhs := rhs.(type) {
		case int64:
			return intCompare(op, lhs, rhs, c.Pos)
		case uint64:
			return intCompare(op, lhs, int64(rhs), c.Pos)
		default:
			return nil, lexer.Errorf(c.Pos, "rhs of %s must be an integer", op)
		}
	case string:
		rhs, ok := rhs.(string)
		if !ok {
			return nil, lexer.Errorf(c.Pos, "rhs of %s must be a string", op)
		}
		return stringCompare(op, lhs, rhs, c.Pos)
	default:
		return nil, lexer.Errorf(c.Pos, "lhs of %s must be an integer or string", op)
	}
}

func (t *Term) Evaluate(instance *Instance) (interface{}, error) {
	lhs, err := t.Unary.Evaluate(instance)
	if err != nil {
		return nil, err
	}

	if t.Op == nil {
		return lhs, nil
	}

	if t.Next == nil {
		return nil, lexer.Errorf(t.Pos, "expected rhs in binary bit operation")
	}

	rhs, err := t.Next.Evaluate(instance)
	if err != nil {
		return nil, err
	}

	op := *t.Op

	switch lhs := lhs.(type) {
	case uint64:
		switch rhs := rhs.(type) {
		case uint64:
			return uintBinaryOp(op, lhs, rhs, t.Pos)
		case int64:
			return uintBinaryOp(op, lhs, uint64(rhs), t.Pos)
		default:
			return nil, lexer.Errorf(t.Pos, `rhs of %s must be an integer`, op)
		}
	case int64:
		switch rhs := rhs.(type) {
		case int64:
			return intBinaryOp(op, lhs, rhs, t.Pos)
		case uint64:
			return intBinaryOp(op, lhs, int64(rhs), t.Pos)
		default:
			return nil, lexer.Errorf(t.Pos, `rhs of %s must be an integer`, op)
		}
	case string:
		switch rhs := rhs.(type) {
		case string:
			return stringBinaryOp(op, lhs, rhs, t.Pos)
		default:
			return nil, lexer.Errorf(t.Pos, "rhs of %s must be a string", op)
		}
	default:
		return nil, lexer.Errorf(t.Pos, "binary bit operation not supported for this type")
	}
}

func (u *Unary) Evaluate(instance *Instance) (interface{}, error) {
	if u.Value != nil {
		return u.Value.Evaluate(instance)
	}

	if u.Unary == nil || u.Op == nil {
		return nil, lexer.Errorf(u.Pos, "invalid unary operation")
	}

	rhs, err := u.Unary.Evaluate(instance)
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
		switch rhs := rhs.(type) {
		case int64:
			return -rhs, nil
		case uint64:
			return -int64(rhs), nil
		default:
			return nil, lexer.Errorf(u.Pos, "rhs of %s must be an integer", *u.Op)
		}
	case "^":
		switch rhs := rhs.(type) {
		case int64:
			return ^rhs, nil
		case uint64:
			return ^rhs, nil
		default:
			return nil, lexer.Errorf(u.Pos, "rhs of %s must be an integer", *u.Op)
		}
	default:
		return nil, lexer.Errorf(u.Pos, "unsupported unary operator %s", *u.Op)
	}
}

func (v *Value) Evaluate(instance *Instance) (interface{}, error) {
	switch {
	case v.Hex != nil:
		return strconv.ParseUint(*v.Hex, 0, 64)
	case v.Octal != nil:
		return strconv.ParseUint(*v.Octal, 8, 64)
	case v.Decimal != nil:
		return *v.Decimal, nil
	case v.String != nil:
		return *v.String, nil
	case v.Variable != nil:
		var (
			ok    bool
			value interface{}
		)
		if instance.Vars != nil {
			value, ok = instance.Vars[*v.Variable]
		}
		if !ok {
			return nil, lexer.Errorf(v.Pos, `unknown variable "%s"`, *v.Variable)
		}
		return coerceIntegers(value), nil
	case v.Subexpression != nil:
		return v.Subexpression.Evaluate(instance)
	case v.Call != nil:
		return v.Call.Evaluate(instance)
	}

	return nil, lexer.Errorf(v.Pos, `unsupported value type "%s"`, repr.String(v))
}

func (a *Array) Evaluate(instance *Instance) (interface{}, error) {
	if a.Ident != nil {
		value, ok := instance.Vars[*a.Ident]
		if !ok {
			return nil, lexer.Errorf(a.Pos, `unknown variable "%s" used as array`, *a.Ident)
		}
		return value, nil
	}
	var result []interface{}
	for _, value := range a.Values {
		v, err := value.Evaluate(instance)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

func (c *Call) Evaluate(instance *Instance) (interface{}, error) {
	var (
		fn Function
		ok bool
	)
	if instance.Functions != nil {
		fn, ok = instance.Functions[c.Name]
	}
	if !ok {
		return nil, lexer.Errorf(c.Pos, `unknown function "%s()"`, c.Name)
	}
	args := []interface{}{}
	for _, arg := range c.Args {
		value, err := arg.Evaluate(instance)
		if err != nil {
			return nil, err
		}
		args = append(args, value)
	}

	value, err := fn(instance, args...)
	if err != nil {
		return nil, lexer.Errorf(c.Pos, `call to "%s()" failed`, c.Name)
	}

	return coerceIntegers(value), nil
}
