package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvalFunction(t *testing.T) {
	assert := assert.New(t)
	expr, err := ParseExpression(`ping("pong") == "pong"`)
	assert.NoError(err)
	assert.NotNil(expr)

	ping := func(args ...interface{}) (interface{}, error) {
		return args[0].(string), nil
	}

	ctx := &Context{
		Functions: map[string]Function{
			"ping": ping,
		},
	}
	value, err := expr.Evaluate(ctx)
	assert.NoError(err)
	assert.Equal(true, value)
}

func TestEvalOctal(t *testing.T) {
	assert := assert.New(t)
	expr, err := ParseExpression(`0644`)
	assert.NoError(err)
	assert.NotNil(expr)

	ctx := &Context{}
	value, err := expr.Evaluate(ctx)
	assert.NoError(err)
	assert.Equal(int64(0644), value)
}

func TestEvalHex(t *testing.T) {
	assert := assert.New(t)
	expr, err := ParseExpression(`0xff`)
	assert.NoError(err)
	assert.NotNil(expr)

	ctx := &Context{}
	value, err := expr.Evaluate(ctx)
	assert.NoError(err)
	assert.Equal(int64(0xff), value)
}

func TestEvalFilePermissions(t *testing.T) {
	assert := assert.New(t)
	expr, err := ParseExpression(`(file.permissions & 0644) == file.permissions`)
	assert.NoError(err)
	assert.NotNil(expr)

	ctx := &Context{
		Vars: map[string]interface{}{
			"file.permissions": int64(044),
		},
	}
	value, err := expr.Evaluate(ctx)
	assert.NoError(err)
	assert.NotNil(value)
}

func TestEvalArrayOperation(t *testing.T) {
	assert := assert.New(t)
	expr, err := ParseExpression(`"abc" in ["abc", "def", 0]`)
	assert.NoError(err)
	assert.NotNil(expr)

	ctx := &Context{}
	value, err := expr.Evaluate(ctx)
	assert.NoError(err)
	assert.Equal(true, value)
}

type testIterable struct {
	contexts []*Context
	index    int
}

func (i *testIterable) Next() (*Context, error) {
	if !i.Done() {
		result := i.contexts[i.index]
		i.index++
		return result, nil
	}
	return nil, errors.New("out of bounds iteration")
}

func (i *testIterable) Done() bool {
	return i.index >= len(i.contexts)
}

func TestEvalIterable(t *testing.T) {
	assert := assert.New(t)
	expr, err := ParseIterable(`len(has("important-property") || file.permissions == 0644) == 2`)
	assert.NoError(err)
	assert.NotNil(expr)

	iterable := &testIterable{
		contexts: []*Context{
			{
				Functions: map[string]Function{
					"has": func(args ...interface{}) (interface{}, error) {
						return true, nil
					},
				},
				Vars: map[string]interface{}{
					"file.permissions": int64(0677),
				},
			},
			{
				Functions: map[string]Function{
					"has": func(args ...interface{}) (interface{}, error) {
						return false, nil
					},
				},
				Vars: map[string]interface{}{
					"file.permissions": int64(0644),
				},
			},
			{
				Functions: map[string]Function{
					"has": func(args ...interface{}) (interface{}, error) {
						return false, nil
					},
				},
				Vars: map[string]interface{}{
					"file.permissions": int64(0),
				},
			},
		},
	}

	value, err := expr.Evaluate(iterable)
	assert.NoError(err)
	assert.Equal(true, value)
}
