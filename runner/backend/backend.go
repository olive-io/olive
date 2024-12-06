package backend

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("key not found")
)

type KV struct {
	Key     []byte
	Value   []byte
	Version uint64
}

type Response struct {
	Kvs []*KV
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

type IBackend interface {
	Get(ctx context.Context, key string, opts ...GetOption) (*Response, error)
	Watch(ctx context.Context, key string, opts ...WatchOption) (Watcher, error)

	Put(ctx context.Context, key string, value any, opts ...PutOption) error
	Del(ctx context.Context, key string, opts ...DelOption) error

	ForceSync() error

	Close() error
}

type GetOptions struct {
	HasPrefix bool
	Reserve   bool
}

type GetOption func(opts *GetOptions)

func GetPrefix() GetOption {
	return func(opts *GetOptions) {
		opts.HasPrefix = true
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
