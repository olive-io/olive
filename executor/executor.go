// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executor

import (
	"context"

	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/server"
)

type Executor struct {
	server.Inner

	ctx    context.Context
	cancel context.CancelFunc

	cfg Config

	oct *client.Client
}

func NewExecutor(cfg Config) (*Executor, error) {
	lg := cfg.Logger
	inner := server.NewInnerServer(lg)

	oct, err := client.New(cfg.Config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	executor := &Executor{
		Inner:  inner,
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
		oct:    oct,
	}

	return executor, nil
}

func (e *Executor) Start() error {

	e.Destroy(e.destroy)
	return nil
}

func (e *Executor) destroy() {

}

func (e *Executor) Stop() {
	return
}
