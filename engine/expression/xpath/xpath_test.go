package xpath

import (
	"context"
	"testing"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/data"
	"github.com/oliveio/olive/engine/expression"
	"github.com/stretchr/testify/assert"
)

func TestXPath(t *testing.T) {
	var engine expression.Engine = New(context.Background())
	compiled, err := engine.CompileExpression("a > 1")
	assert.Nil(t, err)
	result, err := engine.EvaluateExpression(compiled, map[string]interface{}{
		"a": 2,
	})
	assert.Nil(t, err)
	assert.True(t, result.(bool))
}

type dataObjects map[string]data.ItemAware

func (d dataObjects) FindItemAwareById(id bpmn.IdRef) (itemAware data.ItemAware, found bool) {
	itemAware, found = d[id]
	return
}

func (d dataObjects) FindItemAwareByName(name string) (itemAware data.ItemAware, found bool) {
	itemAware, found = d[name]
	return
}

func TestXPath_getDataObject(t *testing.T) {
	// This funtionality doesn't quite work yet
	t.SkipNow()
	var engine = New(context.Background())
	container := data.NewContainer(context.Background(), nil)
	container.Put(context.Background(), data.XMLSource(`<tag attr="val"/>`))
	var objs dataObjects = map[string]data.ItemAware{
		"dataObject": container,
	}
	engine.SetItemAwareLocator(objs)
	compiled, err := engine.CompileExpression("(getDataObject('dataObject')/tag/@attr/string())[1]")
	assert.Nil(t, err)
	result, err := engine.EvaluateExpression(compiled, map[string]interface{}{})
	assert.Nil(t, err)
	assert.Equal(t, "val", result.(string))
}
