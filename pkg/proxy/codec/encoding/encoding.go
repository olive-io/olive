package encoding

import (
	"github.com/olive-io/olive/pkg/proxy/codec"
	"github.com/olive-io/olive/pkg/proxy/codec/bytes"
	"github.com/olive-io/olive/pkg/proxy/codec/json"
	"github.com/olive-io/olive/pkg/proxy/codec/proto"
)

func init() {
	RegisterMarshaler("bytes", &bytes.Marshaler{})
	RegisterMarshaler("json", &json.Marshaler{})
	RegisterMarshaler("proto", &proto.Marshaler{})
}

// mSet the set of Marshaler
var mSet = map[string]codec.Marshaler{}

// RegisterMarshaler puts the implement of Marshaler to mSet
func RegisterMarshaler(name string, m codec.Marshaler) {
	mSet[name] = m
}

// GetMarshaler gets implement from mSet
func GetMarshaler(name string) (codec.Marshaler, bool) {
	m, ok := mSet[name]
	return m, ok
}
