/*
   Copyright 2024 The olive Authors

   This program is offered under a commercial and under the AGPL license.
   For AGPL licensing, see below.

   AGPL licensing:
   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package gateway

import (
	"path"
	"regexp"

	"github.com/cockroachdb/errors"
	iradix "github.com/hashicorp/go-immutable-radix"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/gateway/consumer"
)

var (
	ErrConsumerExists    = errors.New("consumer exists")
	ErrConsumerUnknown   = errors.New("consumer unknown")
	ErrConsumerNotExists = errors.New("consumer does not exist")
)

type hall struct {
	tree *iradix.Tree
}

func createHall() *hall {
	h := &hall{
		tree: iradix.New(),
	}
	return h
}

func (h *hall) get(key string) (consumer.IConsumer, error) {
	c, ok := h.tree.Get([]byte(key))
	if !ok {
		return nil, errors.Wrapf(ErrConsumerExists, "key: %s", key)
	}
	return c.(consumer.IConsumer), nil
}

func (h *hall) find(key string) (consumer.IConsumer, error) {
	var c consumer.IConsumer
	h.tree.Root().Walk(func(k []byte, v interface{}) bool {
		if string(k) == key {
			c = v.(consumer.IConsumer)
			return true
		}
		return false
	})
	if c == nil {
		return nil, errors.Wrapf(ErrConsumerNotExists, "key: %s", key)
	}
	return c, nil
}

func (h *hall) walkPrefix(prefix string, fn func(k string, v consumer.IConsumer) bool) {
	h.tree.Root().WalkPrefix([]byte(prefix), func(k []byte, v interface{}) bool {
		return fn(string(k), v.(consumer.IConsumer))
	})
}

func (h *hall) match(pattern string) (consumer.IConsumer, error) {
	var c consumer.IConsumer
	h.tree.Root().Walk(func(k []byte, v interface{}) bool {
		re := regexp.MustCompile(pattern)
		if matched := re.Match(k); matched {
			c = v.(consumer.IConsumer)
			return true
		}
		return false
	})

	if c == nil {
		return nil, errors.Wrapf(ErrConsumerNotExists, "pattern: %s", pattern)
	}
	return c, nil
}

func (h *hall) insert(key string, cs consumer.IConsumer) error {
	tr, _, exists := h.tree.Insert([]byte(key), cs)
	if exists {
		return errors.Wrapf(ErrConsumerExists, "key %s", key)
	}
	h.tree = tr
	return nil
}

func (h *hall) delete(key string) error {
	tr, _, ok := h.tree.Delete([]byte(key))
	if !ok {
		return errors.Wrapf(ErrConsumerNotExists, "key %s", key)
	}
	h.tree = tr
	return nil
}

func (h *hall) update(key string, cs consumer.IConsumer) error {
	if err := h.delete(key); err != nil {
		return err
	}
	return h.insert(key, cs)
}

func (g *Gateway) AddHandlerWrapper(wrapper consumer.HandlerWrapper) {
	g.handlerWrappers = append(g.handlerWrappers, wrapper)
}

func (g *Gateway) AddConsumer(c consumer.IConsumer, opts ...AddOption) error {
	var options AddOptions
	for _, opt := range opts {
		opt(&options)
	}

	var paths string
	if idt := options.identity; idt != nil {
		paths = consumerPath(idt)
	}
	if paths == "" {
		if rc, ok := c.(consumer.IRegularConsumer); ok {
			paths = consumerPath(rc.Identity())
		}
	}
	if paths == "" {
		paths = options.path
	}

	return g.addConsumer(paths, c)
}

func (g *Gateway) addConsumer(name string, c consumer.IConsumer) error {
	if name == "" {
		return ErrConsumerUnknown
	}

	g.hmu.Lock()
	defer g.hmu.Unlock()
	return g.hall.insert(name, c)
}

func (g *Gateway) walkConsumer(prefix string, fn func(key string, c consumer.IConsumer) bool) {
	g.hmu.RLock()
	defer g.hmu.RUnlock()
	g.hall.walkPrefix(prefix, fn)
}

func consumerPath(idt *dsypb.Consumer) string {
	paths := "/"
	paths = path.Join(paths, idt.Activity.String())
	if idt.Action != "" {
		paths = path.Join(paths, idt.Action)
	}
	if idt.Id != "" {
		paths = path.Join(paths, idt.Id)
	}
	return paths
}
