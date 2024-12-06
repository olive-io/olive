package controller

import (
	"github.com/olive-io/olive/pkg/queue"
	"github.com/olive-io/olive/runner/backend"
)

type Controller struct {
	db backend.IBackend
	// inner process queue
	pq queue.SyncPriorityQueue[*Process]
}

func (c *Controller) process() {

}
