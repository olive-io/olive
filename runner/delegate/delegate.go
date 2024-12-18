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

package delegate

import (
	"context"
	"errors"
	"sync"
	"time"

	corev1 "github.com/olive-io/olive/api/types/core/v1"
	"github.com/olive-io/olive/runner/metrics"
	"github.com/olive-io/olive/runner/storage"
)

var (
	ErrExists   = errors.New("delegate exists")
	ErrNotFound = errors.New("delegate not found")
)

// Theme is the logo of Delegate
type Theme struct {
	Major corev1.FlowNodeType `json:"major"`

	Minor string `json:"minor"`
}

func (t Theme) String() string {
	return "/" + string(t.Major) + "/" + t.Minor
}

type Request struct {
	Headers     map[string]string
	Properties  map[string]any
	DataObjects map[string]any
	Timeout     time.Duration
}

type Response struct {
	Result      map[string]any
	DataObjects map[string]any
}

type Delegate interface {
	GetTheme() Theme
	Call(ctx context.Context, req *Request) (*Response, error)
}

var (
	router             *Tree = New()
	internalController *Controller
	once               sync.Once
)

func Init(cfg *Config, bs *storage.Storage) error {
	var err error
	once.Do(func() {
		internalController, err = NewControllerWithTree(cfg, router, bs)
		if err != nil {
			return
		}

		features := map[string][]string{}
		router.Walk(func(s string, v interface{}) bool {
			delegate, ok := v.(Delegate)
			if !ok {
				return false
			}

			theme := delegate.GetTheme()
			list, ok := features[string(theme.Major)]
			if !ok {
				list = []string{}
				features[string(theme.Major)] = list
			}
			list = append(list, theme.Minor)
			features[string(theme.Major)] = list

			return false
		})

		for name, feature := range features {
			if err = metrics.AddFeatures(name, feature...); err != nil {
				return
			}
		}
	})

	return err
}

// RegisterDelegate registers a delegate into internal router.
func RegisterDelegate(delegate Delegate) error {
	theme := delegate.GetTheme()
	url := theme.String()

	if _, ok := router.Get(url); ok {
		return ErrExists
	}

	router.Insert(url, delegate)
	return nil
}

// GetDelegate returns a delegate by url from internal router.
func GetDelegate(url string) (Delegate, error) {
	return internalController.GetDelegate(url)
}

// Controller is delegate controller.
type Controller struct {
	cfg *Config

	root *Tree

	bs *storage.Storage
}

func NewController(cfg *Config, bs *storage.Storage) (*Controller, error) {
	return NewControllerWithTree(cfg, New(), bs)
}

func NewControllerWithTree(cfg *Config, tree *Tree, bs *storage.Storage) (*Controller, error) {
	controller := &Controller{
		cfg:  cfg,
		root: tree,
		bs:   bs,
	}

	return controller, nil
}

// Register registers a delegate into Controller.
func (c *Controller) Register(delegate Delegate) error {
	theme := delegate.GetTheme()
	url := theme.String()

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
