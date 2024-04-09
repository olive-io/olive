/*
   Copyright 2023 The olive Authors

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
