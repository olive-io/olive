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

type IBackend interface {
	Get(ctx context.Context, key string, opts ...GetOption) (*Response, error)
	Set(ctx context.Context, key string, value any, opts ...SetOption) error
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

type SetOptions struct {
	TTL time.Duration
}

type SetOption func(opts *SetOptions)

func SetDuration(ttl time.Duration) SetOption {
	return func(opts *SetOptions) {
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
