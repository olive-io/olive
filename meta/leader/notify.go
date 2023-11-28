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

package leader

import (
	"go.etcd.io/etcd/server/v3/etcdserver"
)

type Notifier interface {
	ChangeNotify() <-chan struct{}
	ReadyNotify() <-chan struct{}
	IsLeader() bool
}

type notifier struct {
	s *etcdserver.EtcdServer
}

func NewNotify(s *etcdserver.EtcdServer) Notifier {
	return &notifier{s: s}
}

func (n *notifier) ChangeNotify() <-chan struct{} {
	return n.s.LeaderChangedNotify()
}

func (n *notifier) ReadyNotify() <-chan struct{} {
	return n.s.ReadyNotify()
}

func (n *notifier) IsLeader() bool {
	lead := n.s.Leader()
	return n.s.ID() == lead && lead != 0
}
