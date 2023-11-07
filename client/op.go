package client

type opType int

const (
// A default Op has opType 0, which is invalid
)

// Op represents an Operation that kv can execute
type Op struct {
	t opType
}

type OpResponse struct{}
