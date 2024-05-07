package openapi

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTesters tests the testers helpers
// that determines the kind of a type.
func TestTesters(t *testing.T) {
	var (
		s  string
		m  map[string]string
		n1 uint
		n2 int
	)
	assert.True(t, isString(rt(s)))
	assert.True(t, isMap(rt(m)))
	assert.True(t, isNumber(rt(n1)))
	assert.True(t, isNumber(rt(n2)))
	assert.False(t, isNumber(rt(s)))
}

// TestSchemaValidation tests that the validator.v8
// tags are properly translated to JSON Schema validation
// fields for all supported types.
func TestSchemaValidation(t *testing.T) {
	type T struct {
		A string            `validate:"len=12"`
		B int               `validate:"min=5,max=100"`
		C []bool            `validate:"len=50"`
		D map[string]string `validate:"eq=5"`
		E string            `validate:"eq=15"`
		F string            `validate:"lt=8"`
		G string            `validate:"gt=6"`
		H int               `validate:"len=foo"`    // ignored, cannot convert "foo" to integer
		I []string          `validate:"dive,len=6"` // ignored, dive option present before tag
		J map[string]string `validate:"keys,max=6"` // ignored, keys option present before tag
	}
	g := gen(t)

	sor := g.newSchemaFromType(rt(new(T)))
	assert.NotNil(t, sor)
	schema := g.resolveSchema(sor)
	assert.NotNil(t, schema)

	actual, err := json.Marshal(schema)
	if err != nil {
		t.Error(err)
	}
	// see testdata/validation/len.json.
	expected, err := os.ReadFile("../testdata/schemas/validation.json")
	if err != nil {
		t.Error(err)
	}
	m, err := diffJSON(actual, expected)
	if err != nil {
		t.Error(err)
	}
	if !m {
		t.Error("expected json outputs to be equal")
	}
}
