/*
Copyright 2025 The olive Authors

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

package delegate

import (
	"context"
	"errors"
	"time"
)

var (
	ErrExists   = errors.New("delegate exists")
	ErrNotFound = errors.New("delegate not found")
)

type Request struct {
	Headers     map[string]string
	Properties  map[string]string
	DataObjects map[string]string
	Timeout     time.Duration
}

type Response struct {
	Result      map[string]string
	DataObjects map[string]string
}

type Delegate interface {
	Commit(ctx context.Context, req *Request) (*Response, error)
	Rollback(ctx context.Context) error
	Destroy(ctx context.Context) error
}

// Controller is delegate controller.
type Controller struct {
	root *Tree
}

func NewController() (*Controller, error) {
	return NewControllerWithTree(New())
}

func NewControllerWithTree(tree *Tree) (*Controller, error) {
	controller := &Controller{
		root: tree,
	}

	return controller, nil
}

// Register registers a delegate into Controller.
func (c *Controller) Register(url string, delegate Delegate) error {
	if _, ok := c.root.Get(url); ok {
		return ErrExists
	}

	c.root.Insert(url, delegate)
	return nil
}

// GetDelegate returns a delegate by url.
func (c *Controller) GetDelegate(url string) (Delegate, error) {
	value, ok := c.root.Get(url)
	if !ok {
		return nil, ErrNotFound
	}
	return value.(Delegate), nil
}
