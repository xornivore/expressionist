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
	contexts := []*Context{
		{
			Functions: map[string]Function{
				"has": func(args ...interface{}) (interface{}, error) {
					return true, nil
				},
			},
			Vars: map[string]interface{}{
				"file.permissions": int64(0677),
				"file.owner":       "root",
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
				"file.owner":       "root",
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
				"file.owner":       "root",
			},
		},
	}

	tests := []struct {
		name       string
		expression string
		result     bool
	}{
		{
			name:       "len",
			expression: `len(has("important-property") || file.permissions == 0644) == 2`,
			result:     true,
		},
		{
			name:       "all",
			expression: `all(file.owner == "root")`,
			result:     true,
		},
		{
			name:       "none",
			expression: `none(file.owner == "alice")`,
			result:     true,
		},
		{
			name:       "any",
			expression: `any(file.owner == "alice")`,
			result:     false,
		},
		{
			name:       "no function",
			expression: `file.owner == "alice"`,
			result:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			iterable := &testIterable{
				contexts: contexts,
			}

			assert := assert.New(t)
			expr, err := ParseIterable(test.expression)
			assert.NoError(err)
			assert.NotNil(expr)

			value, err := expr.Evaluate(iterable)
			assert.NoError(err)
			assert.Equal(test.result, value)
		})
	}
}

func TestEvalPathExpression(t *testing.T) {
	ctx := &Context{
		Functions: map[string]Function{
			"shell.command.stdout": func(args ...interface{}) (interface{}, error) {
				return "/etc/path-from-command", nil
			},
			"process.flag": func(args ...interface{}) (interface{}, error) {
				return "/etc/path-from-process", nil
			},
		},
	}

	tests := []struct {
		name       string
		expression string
		path       string
	}{
		{
			name:       "path",
			expression: `/etc/passwd`,
			path:       `/etc/passwd`,
		},
		{
			name:       "glob",
			expression: `/var/run/*.sock`,
			path:       `/var/run/*.sock`,
		},
		{
			name:       "path from command",
			expression: `shell.command.stdout("/usr/bin/find-my-path", "-v")`,
			path:       "/etc/path-from-command",
		},
		{
			name:       "path from process flag",
			expression: `process.flag("kubelet", "--config")`,
			path:       "/etc/path-from-process",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			assert := assert.New(t)
			expr, err := ParsePath(test.expression)
			assert.NoError(err)
			assert.NotNil(expr)

			value, err := expr.Evaluate(ctx)
			assert.NoError(err)
			assert.Equal(test.path, value)
		})
	}
}
