package datadir

import "path/filepath"

const (
	snapDirSegment     = "snap"
	walDirSegment      = "wal"
	backendFileSegment = "db"
)

func ToBackendFileName(dataDir string) string {
	return filepath.Join(dataDir, backendFileSegment)
}

func ToSnapDir(dataDir string) string {
	return filepath.Join(dataDir, snapDirSegment)
}

func ToWalDir(dataDir string) string {
	return filepath.Join(dataDir, walDirSegment)
}
