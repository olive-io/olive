package backend

import (
	"context"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/dgraph-io/badger/v4"
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
		options.Logger = &Logger{cfg.Logger}
	}

	db, err := badger.Open(options)
	if err != nil {
		return nil, err
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
			kv := itemToKV(item)
			resp.Kvs = append(resp.Kvs, kv)
		}
	} else {
		item, err := tx.Get(keyBytes)
		if err != nil {
			return nil, err
		}
		kv := itemToKV(item)
		resp.Kvs = append(resp.Kvs, kv)
	}

	return resp, nil
}

func (b *backend) Set(ctx context.Context, key string, value any, opts ...SetOption) error {
	var options SetOptions
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
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
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
				return err
			}
		}

	} else {
		if err := tx.Delete([]byte(key)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (b *backend) ForceSync() error {
	return b.db.Sync()
}

func (b *backend) Close() error {
	return b.db.Close()
}

func itemToKV(item *badger.Item) *KV {
	kv := &KV{}
	kv.Key = item.Key()
	kv.Value, _ = item.ValueCopy(nil)
	kv.Version = item.Version()
	return kv
}
