package runner

import (
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/server"
)

type Runner struct {
	Config

	osv    *server.OliveServer
	client *client.Client

	done  chan struct{}
	close chan struct{}
}

func (r *Runner) Start() error {
	return nil
}

func (r *Runner) run() {}
