package storage

import (
	"bytes"
	"context"
	"errors"
	"time"

	json "github.com/bytedance/sonic"
)

var (
	ErrNotFound   = errors.New("key not found")
	ErrConflict   = errors.New("transaction Conflict. Please retry")
	ErrEmptyKey   = errors.New("key is empty")
	ErrEmptyValue = errors.New("value is empty")
)

type KV struct {
	Key     []byte
	Value   []byte
	Version uint64
}

type Response struct {
	Kvs []*KV
}

func (r *Response) Unmarshal(v any) error {
	if len(r.Kvs) == 0 {
		return ErrEmptyValue
	}
	return json.Unmarshal(r.Data(), v)
}

func (r *Response) Data() []byte {
	if len(r.Kvs) == 0 {
		return []byte("")
	}
	if len(r.Kvs) == 1 {
		return r.Kvs[0].Value
	}
	buf := bytes.NewBufferString("[")
	for i, kv := range r.Kvs {
		buf.Write(kv.Value)
		if i != len(r.Kvs)-1 {
			buf.WriteString(",")
		}
	}
	buf.WriteString("]")
	return buf.Bytes()
}

type EventOp int

const (
	OpPut EventOp = iota + 1
	OpDel
)

func (op EventOp) String() string {
	switch op {
	case OpPut:
		return "put"
	case OpDel:
		return "del"
	default:
		return "unknown"
	}
}

type Event struct {
	Op EventOp
	Kv *KV
}

type Watcher interface {
	Next() (*Event, error)
}

type Storage interface {
	Get(ctx context.Context, key string, opts ...GetOption) (*Response, error)
	Watch(ctx context.Context, key string, opts ...WatchOption) (Watcher, error)

	Put(ctx context.Context, key string, value any, opts ...PutOption) error
	Del(ctx context.Context, key string, opts ...DelOption) error

	ForceSync() error

	Close() error
}

type GetOptions struct {
	HasPrefix bool
	Seek      string
	Reserve   bool
}

type GetOption func(opts *GetOptions)

func GetPrefix() GetOption {
	return func(opts *GetOptions) {
		opts.HasPrefix = true
	}
}

func GetSeek(firstKey string) GetOption {
	return func(opts *GetOptions) {
		opts.Seek = firstKey
	}
}

func GetReserve() GetOption {
	return func(opts *GetOptions) {
		opts.Reserve = true
	}
}

type WatchOptions struct {
}

type WatchOption func(opts *WatchOptions)

type PutOptions struct {
	TTL time.Duration
}

type PutOption func(opts *PutOptions)

func PutDuration(ttl time.Duration) PutOption {
	return func(opts *PutOptions) {
		opts.TTL = ttl
	}
}

type DelOptions struct {
	HasPrefix bool
}

type DelOption func(opts *DelOptions)

func DelPrefix() DelOption {
	return func(opts *DelOptions) {
		opts.HasPrefix = true
	}
}
