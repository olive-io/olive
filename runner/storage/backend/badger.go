package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/pb"
	"go.uber.org/zap"
)

const (
	DefaultCacheSize  = 5 * 1024 * 1024
	DefaultGCInterval = 5 * time.Minute
)

type Config struct {
	Dir       string
	CacheSize int64

	GCInterval time.Duration
	Logger     *zap.Logger
}

func NewConfig(dir string) *Config {
	cfg := &Config{
		Dir:        dir,
		CacheSize:  DefaultCacheSize,
		GCInterval: DefaultGCInterval,
	}

	return cfg
}

type backend struct {
	cfg *Config

	db *badger.DB
}

func NewBackend(cfg *Config) (IBackend, error) {
	options := badger.DefaultOptions(cfg.Dir)
	options.BlockCacheSize = cfg.CacheSize
	if cfg.Logger != nil {
		logger := cfg.Logger.WithOptions(zap.AddCallerSkip(2))
		options.Logger = &Logger{logger}
	}

	db, err := badger.Open(options)
	if err != nil {
		return nil, parseBadgerErr(err)
	}

	b := &backend{
		cfg: cfg,
		db:  db,
	}

	go b.process()

	return b, nil
}

func (b *backend) process() {
	duration := b.cfg.GCInterval
	if duration <= 0 {
		duration = DefaultGCInterval
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			_ = b.db.RunValueLogGC(0.7)
		}
	}
}

func (b *backend) Get(ctx context.Context, key string, opts ...GetOption) (*Response, error) {
	var options GetOptions
	for _, opt := range opts {
		opt(&options)
	}

	keyBytes := []byte(key)
	resp := &Response{Kvs: make([]*KV, 0)}

	tx := b.db.NewTransaction(false)
	defer tx.Discard()

	if options.HasPrefix {
		iterOptions := badger.DefaultIteratorOptions
		iterOptions.Reverse = options.Reserve

		iter := tx.NewIterator(iterOptions)
		defer iter.Close()
		prefix := keyBytes

		for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
			item := iter.Item()
			if item.IsDeletedOrExpired() {
				continue
			}
			kv := itemToKV(item)
			resp.Kvs = append(resp.Kvs, kv)
		}
	} else {
		item, err := tx.Get(keyBytes)
		if err != nil {
			return nil, parseBadgerErr(err)
		}
		if item.IsDeletedOrExpired() {
			return nil, ErrNotFound
		}
		kv := itemToKV(item)
		resp.Kvs = append(resp.Kvs, kv)
	}

	return resp, nil
}

func (b *backend) Watch(ctx context.Context, key string, opts ...WatchOption) (Watcher, error) {
	var options WatchOptions
	for _, opt := range opts {
		opt(&options)
	}

	matches := make([]pb.Match, 0)
	matches = append(matches, pb.Match{Prefix: []byte(key)})

	wch := make(chan *Event, 10)
	ech := make(chan error, 1)

	cb := func(kvList *badger.KVList) error {
		for _, item := range kvList.Kv {
			kv := &KV{
				Key:     item.Key,
				Value:   item.Value,
				Version: item.Version,
			}
			event := &Event{}
			if len(item.Value) == 0 {
				event.Op = OpDel
			} else {
				event.Op = OpPut
			}
			event.Kv = kv
			wch <- event
		}
		return nil
	}

	go func() {
		err := b.db.Subscribe(ctx, cb, matches)
		if err != nil {
			ech <- parseBadgerErr(err)
		}
	}()

	w := newWatcher(ctx, wch, ech)
	return w, nil
}

func (b *backend) Put(ctx context.Context, key string, value any, opts ...PutOption) error {
	var options PutOptions
	for _, opt := range opts {
		opt(&options)
	}

	tx := b.db.NewTransaction(true)
	defer tx.Discard()

	var data []byte
	switch vv := value.(type) {
	case []byte:
		data = vv
	case *[]byte:
		data = []byte(*vv)
	case string:
		data = []byte(vv)
	case *string:
		data = []byte(*vv)
	default:
		var err error
		data, err = json.Marshal(value)
		if err != nil {
			return err
		}
	}

	entry := badger.NewEntry([]byte(key), data)
	if options.TTL > 0 {
		entry = entry.WithTTL(options.TTL)
	}
	if err := tx.SetEntry(entry); err != nil {
		return parseBadgerErr(err)
	}

	if err := tx.Commit(); err != nil {
		return parseBadgerErr(err)
	}

	return nil
}

func (b *backend) Del(ctx context.Context, key string, opts ...DelOption) error {
	var options DelOptions
	for _, opt := range opts {
		opt(&options)
	}

	tx := b.db.NewTransaction(true)
	defer tx.Discard()

	if options.HasPrefix {
		prefix := []byte(key)
		iterOptions := badger.DefaultIteratorOptions
		it := tx.NewIterator(iterOptions)

		keys := make([][]byte, 0)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			keys = append(keys, it.Item().Key())
		}
		it.Close()

		for _, item := range keys {
			if err := tx.Delete(item); err != nil {
				return parseBadgerErr(err)
			}
		}

	} else {
		if err := tx.Delete([]byte(key)); err != nil {
			return parseBadgerErr(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return parseBadgerErr(err)
	}

	return nil
}

func (b *backend) ForceSync() error {
	return b.db.Sync()
}

func (b *backend) Close() error {
	return b.db.Close()
}

type watcher struct {
	ctx context.Context
	wch chan *Event
	ech chan error
}

func newWatcher(ctx context.Context, wch chan *Event, ech chan error) *watcher {
	w := &watcher{
		ctx: ctx,
		wch: wch,
		ech: ech,
	}

	return w
}

func (w *watcher) Next() (*Event, error) {
	select {
	case <-w.ctx.Done():
		return nil, io.EOF
	case event := <-w.wch:
		return event, nil
	case err := <-w.ech:
		return nil, err
	}
}

func itemToKV(item *badger.Item) *KV {
	kv := &KV{}
	kv.Key = item.Key()
	kv.Value, _ = item.ValueCopy(nil)
	kv.Version = item.Version()
	return kv
}

func parseBadgerErr(err error) error {
	if errors.Is(err, badger.ErrKeyNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, badger.ErrConflict) {
		return ErrConflict
	}
	if errors.Is(err, badger.ErrEmptyKey) {
		return ErrEmptyKey
	}
	return fmt.Errorf("badger: %w", err)
}
