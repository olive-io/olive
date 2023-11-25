// Copyright 2023 Lack (xingyys@gmail.com).
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

package meta

import (
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type scheduler struct {
	v3cli *clientv3.Client

	lg      *zap.Logger
	activeQ string

	stopping <-chan struct{}
}

func (s *Server) newScheduler() (*scheduler, error) {
	sc := &scheduler{
		stopping: s.StoppingNotify(),
	}

	go sc.run()
	return sc, nil
}

func (s *scheduler) schedulingCycle() error {
	return nil
}

func (s *scheduler) run() {
	for {
		select {
		case <-s.stopping:
			return
		}
	}
}
