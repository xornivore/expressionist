package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvalFunction(t *testing.T) {
	assert := assert.New(t)
	expr, err := Parse(`ping("pong") == "pong"`)
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
	expr, err := Parse(`0644`)
	assert.NoError(err)
	assert.NotNil(expr)

	ctx := &Context{}
	value, err := expr.Evaluate(ctx)
	assert.NoError(err)
	assert.Equal(int64(0644), value)
}

func TestEvalHex(t *testing.T) {
	assert := assert.New(t)
	expr, err := Parse(`0xff`)
	assert.NoError(err)
	assert.NotNil(expr)

	ctx := &Context{}
	value, err := expr.Evaluate(ctx)
	assert.NoError(err)
	assert.Equal(int64(0xff), value)
}

func TestEvalFilePermissions(t *testing.T) {
	assert := assert.New(t)
	expr, err := Parse(`(file.permissions & 0644) == file.permissions`)
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
	expr, err := Parse(`"abc" in ["abc", "def", 0]`)
	assert.NoError(err)
	assert.NotNil(expr)

	ctx := &Context{}
	value, err := expr.Evaluate(ctx)
	assert.NoError(err)
	assert.Equal(true, value)
}

func TestEvalListEnum(t *testing.T) {
	assert := assert.New(t)
	expr, err := Parse(`len(jq(".important-property")) == 0`)
	assert.NoError(err)
	assert.NotNil(expr)

	jq := func(args ...interface{}) (interface{}, error) {
		return "blah", nil
	}

	len := func(args ...interface{}) (interface{}, error) {
		return int64(0), nil
	}

	ctx := &Context{
		Functions: map[string]Function{
			"jq":  jq,
			"len": len,
		},
	}
	value, err := expr.Evaluate(ctx)
	assert.NoError(err)
	assert.Equal(true, value)
}
