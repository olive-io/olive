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

	apiErr "github.com/olive-io/olive/api/errors"
)

var (
	globalNotifier Notifier
	readyc         = make(chan struct{}, 1)
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

func InitNotifier(e *etcdserver.EtcdServer) {
	globalNotifier = NewNotify(e)

	go func() {
		select {
		case <-readyc:
			return
		case <-globalNotifier.ReadyNotify():
			close(readyc)
		}
	}()
}

// WaitReadyc returns global channel readyc
func WaitReadyc() <-chan struct{} {
	return readyc
}

func WaitUtilReady() {
	select {
	case <-readyc:
	}
}

// RequestForReadyc returns error if embed etcd not ready
func RequestForReadyc() error {
	select {
	case <-readyc:
		return nil
	default:
		return apiErr.NewInternal("server is not ready for requesting")
	}
}
