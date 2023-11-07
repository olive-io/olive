package serverpb

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
)

// InternalRaftStringer implements custom proto Stringer:
// redact password, replace value fields with value_size fields.
type InternalRaftStringer struct {
	Request *InternalRaftRequest
}

func (as *InternalRaftStringer) String() string {
	switch {
	case as.Request.Put != nil:
		return fmt.Sprintf("header:<%s> put:<%s>",
			as.Request.Header.String(),
			NewLoggablePutRequest(as.Request.Put).String(),
		)
	case as.Request.Txn != nil:
		return fmt.Sprintf("header:<%s> txn:<%s>",
			as.Request.Header.String(),
			NewLoggableTxnRequest(as.Request.Txn).String(),
		)
	default:
		// nothing to redact
	}
	return as.Request.String()
}

// txnRequestStringer implements a custom proto String to replace value bytes fields with value size
// fields in any nested txn and put operations.
type txnRequestStringer struct {
	Request *TxnRequest
}

func NewLoggableTxnRequest(request *TxnRequest) *txnRequestStringer {
	return &txnRequestStringer{request}
}

func (as *txnRequestStringer) String() string {
	var compare []string
	for _, c := range as.Request.Compare {
		switch cv := c.TargetUnion.(type) {
		case *Compare_Value:
			compare = append(compare, newLoggableValueCompare(c, cv).String())
		default:
			// nothing to redact
			compare = append(compare, c.String())
		}
	}
	var success []string
	for _, s := range as.Request.Success {
		success = append(success, newLoggableRequestOp(s).String())
	}
	var failure []string
	for _, f := range as.Request.Failure {
		failure = append(failure, newLoggableRequestOp(f).String())
	}
	return fmt.Sprintf("compare:<%s> success:<%s> failure:<%s>",
		strings.Join(compare, " "),
		strings.Join(success, " "),
		strings.Join(failure, " "),
	)
}

// requestOpStringer implements a custom proto String to replace value bytes fields with value
// size fields in any nested txn and put operations.
type requestOpStringer struct {
	Op *RequestOp
}

func newLoggableRequestOp(op *RequestOp) *requestOpStringer {
	return &requestOpStringer{op}
}

func (as *requestOpStringer) String() string {
	switch op := as.Op.Request.(type) {
	case *RequestOp_RequestPut:
		return fmt.Sprintf("request_put:<%s>", NewLoggablePutRequest(op.RequestPut).String())
	default:
		// nothing to redact
	}
	return as.Op.String()
}

// loggableValueCompare implements a custom proto String for Compare.Value union member types to
// replace the value bytes field with a value size field.
// To preserve proto encoding of the key and range_end bytes, a faked out proto type is used here.
type loggableValueCompare struct {
	Result    Compare_CompareResult `protobuf:"varint,1,opt,name=result,proto3,enum=Compare_CompareResult"`
	Target    Compare_CompareTarget `protobuf:"varint,2,opt,name=target,proto3,enum=Compare_CompareTarget"`
	Key       []byte                `protobuf:"bytes,3,opt,name=key,proto3"`
	ValueSize int64                 `protobuf:"varint,7,opt,name=value_size,proto3"`
	RangeEnd  []byte                `protobuf:"bytes,64,opt,name=range_end,proto3"`
}

func newLoggableValueCompare(c *Compare, cv *Compare_Value) *loggableValueCompare {
	return &loggableValueCompare{
		c.Result,
		c.Target,
		c.Key,
		int64(len(cv.Value)),
		c.RangeEnd,
	}
}

func (m *loggableValueCompare) Reset()         { *m = loggableValueCompare{} }
func (m *loggableValueCompare) String() string { return proto.CompactTextString(m) }
func (*loggableValueCompare) ProtoMessage()    {}

// loggablePutRequest implements a custom proto String to replace value bytes field with a value
// size field.
// To preserve proto encoding of the key bytes, a faked out proto type is used here.
type loggablePutRequest struct {
	Key         []byte `protobuf:"bytes,1,opt,name=key,proto3"`
	ValueSize   int64  `protobuf:"varint,2,opt,name=value_size,proto3"`
	PrevKv      bool   `protobuf:"varint,3,opt,name=prev_kv,proto3"`
	IgnoreValue bool   `protobuf:"varint,4,opt,name=ignore_value,proto3"`
}

func NewLoggablePutRequest(request *PutRequest) *loggablePutRequest {
	return &loggablePutRequest{
		request.Key,
		int64(len(request.Value)),
		request.PrevKv,
		request.IgnoreValue,
	}
}

func (m *loggablePutRequest) Reset()         { *m = loggablePutRequest{} }
func (m *loggablePutRequest) String() string { return proto.CompactTextString(m) }
func (*loggablePutRequest) ProtoMessage()    {}
