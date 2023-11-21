package runner

import (
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/dustin/go-humanize"
	"go.uber.org/zap"
)

var defaultSplit pebble.Split = func(a []byte) int {
	return 1
}

func newStorage(cfg *Config) (*pebble.DB, error) {
	lg := cfg.Logger
	pcache := pebble.NewCache(cfg.CacheSize)

	comparer := pebble.DefaultComparer
	comparer.Split = defaultSplit

	wal := cfg.WALDir()
	_ = os.MkdirAll(wal, os.ModePerm)
	options := &pebble.Options{
		Cache:    pcache,
		WALDir:   cfg.WALDir(),
		Comparer: comparer,
	}

	dir := cfg.DBDir()
	_ = os.MkdirAll(dir, os.ModePerm)
	db, err := pebble.Open(dir, options)
	if err != nil {
		return nil, err
	}

	lg.Debug("open local storage",
		zap.String("cache", humanize.IBytes(uint64(cfg.CacheSize))),
		zap.String("db", dir),
		zap.String("wal", cfg.WALDir()))

	if err = db.Flush(); err != nil {
		return nil, err
	}

	return db, nil
}
