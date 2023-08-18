package bpmn

import (
	"encoding/xml"
	"testing"

	"github.com/oliveio/olive/test"
	"github.com/stretchr/testify/assert"
)

func TestParseSample(t *testing.T) {
	var sampleDoc Definitions
	var err error
	test.LoadTestFile("sample/bpmn/sample.bpmn", &sampleDoc)
	processes := sampleDoc.Processes()
	assert.Equal(t, 1, len(*processes))

	out, err := xml.MarshalIndent(sampleDoc, "", " ")
	if !assert.NoError(t, err) {
		return
	}

	t.Log(string(out))
}

func TestParseSampleNs(t *testing.T) {
	var sampleDoc Definitions
	test.LoadTestFile("sample/bpmn/sample_ns.bpmn", &sampleDoc)
	processes := sampleDoc.Processes()
	assert.Equal(t, 1, len(*processes))
}
