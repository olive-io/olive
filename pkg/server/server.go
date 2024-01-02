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

package server

import (
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

type Inner interface {
	// StopNotify returns a channel that receives an empty struct
	// when the server is stopped.
	StopNotify() <-chan struct{}
	// StoppingNotify returns a channel that receives an empty struct
	// when the server is being stopped.
	StoppingNotify() <-chan struct{}
	// GoAttach creates a goroutine on a given function and tracks it using the waitgroup.
	// The passed function should interrupt on s.StoppingNotify().
	GoAttach(fn func())
	// Destroy run destroy function when the server stop
	Destroy(fn func())
	// Shutdown sends signal to stop channel and all goroutines stop
	Shutdown()
}

type innerServer struct {
	lg *zap.Logger

	stopping chan struct{}
	done     chan struct{}
	stop     chan struct{}

	wgMu sync.RWMutex
	wg   sync.WaitGroup
}

func NewInnerServer(lg *zap.Logger) Inner {
	s := &innerServer{
		lg:       lg,
		stopping: make(chan struct{}, 1),
		done:     make(chan struct{}, 1),
		stop:     make(chan struct{}, 1),
		wgMu:     sync.RWMutex{},
		wg:       sync.WaitGroup{},
	}

	return s
}

func (s *innerServer) StopNotify() <-chan struct{} { return s.done }

func (s *innerServer) StoppingNotify() <-chan struct{} { return s.stopping }

func (s *innerServer) GoAttach(fn func()) {
	s.wgMu.RLock() // this blocks with ongoing close(s.stopping)
	defer s.wgMu.RUnlock()
	select {
	case <-s.stopping:
		s.lg.Warn("server has stopped; skipping GoAttach")
		return
	default:
	}

	// now safe to add since waitgroup wait has not started yet
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		fn()
	}()
}

func (s *innerServer) Destroy(fn func()) {
	go s.destroy(fn)
}

func (s *innerServer) destroy(fn func()) {
	defer func() {
		s.wgMu.Lock() // block concurrent waitgroup adds in GoAttach while stopping
		close(s.stopping)
		s.wgMu.Unlock()

		s.wg.Wait()

		// clean something
		s.lg.Debug("server has stopped, running destroy operations")
		fn()

		close(s.done)
	}()

	<-s.stop
}

func (s *innerServer) Shutdown() {
	select {
	case s.stop <- struct{}{}:
	case <-s.done:
		return
	}
	<-s.done
}

// GRPCHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Given in gRPC docs.
func GRPCHandlerFunc(gh *grpc.Server, hh http.Handler) http.Handler {
	h2s := &http2.Server{}
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			gh.ServeHTTP(w, r)
		} else {
			hh.ServeHTTP(w, r)
		}
	}), h2s)
}
