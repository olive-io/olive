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

package http

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"

	pb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/runner/internal/client"
	"github.com/olive-io/olive/runner/internal/client/http/test"
	"github.com/olive-io/olive/runner/internal/client/selector"
	"go.etcd.io/etcd/server/v3/embed"
	"go.etcd.io/etcd/server/v3/etcdserver"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
)

func newEtcd(t *testing.T) (*etcdserver.EtcdServer, func()) {
	cfg := embed.NewConfig()
	cfg.Dir = "testdata"
	etcdServer, err := embed.StartEtcd(cfg)
	if err != nil {
		t.Fatal(err)
	}

	<-etcdServer.Server.ReadyNotify()

	cancel := func() {
		etcdServer.Server.HardStop()
		<-etcdServer.Server.StopNotify()
		_ = os.RemoveAll("testdata")
	}
	return etcdServer.Server, cancel
}

func TestHTTPClient(t *testing.T) {
	etcdServer, cancel := newEtcd(t)
	defer cancel()

	v3cli := v3client.New(etcdServer)

	dsy, _ := discovery.NewDiscovery(v3cli)
	s, _ := selector.NewSelector(selector.Discovery(dsy))

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/foo/bar", func(w http.ResponseWriter, r *http.Request) {
		// only accept post
		if r.Method != "POST" {
			http.Error(w, "expect post method", 500)
			return
		}

		// get codec
		ct := r.Header.Get("Content-Type")
		codec, ok := defaultHTTPCodecs[ct]
		if !ok {
			http.Error(w, "codec not found", 500)
			return
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// extract message
		msg := new(test.Message)
		if err := codec.Unmarshal(b, msg); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// marshal response
		b, err = codec.Marshal(msg)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// write response
		w.Write(b)
	})
	go http.Serve(l, mux)

	if err := dsy.Register(context.TODO(), &pb.Service{
		Name: "test.service",
		Nodes: []*pb.Node{
			{
				Id:      "test.service.1",
				Address: l.Addr().String(),
				Metadata: map[string]string{
					"protocol": "http",
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(client.Discovery(dsy), client.Selector(s))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		msg := &test.Message{
			Seq:  int64(i),
			Data: fmt.Sprintf("message %d", i),
		}
		endpoint := "/foo/bar"
		if i%2 == 0 {
			endpoint = endpoint + "?pageNum=1&pageSize=2"
		}
		req := c.NewRequest("test.service", endpoint, msg)
		rsp := new(test.Message)
		err := c.Call(context.TODO(), req, rsp)
		if err != nil {
			t.Fatal(err)
		}
		if rsp.Seq != msg.Seq {
			t.Fatalf("invalid seq %d for %d", rsp.Seq, msg.Seq)
		}
	}
}

func TestHTTPClientStream(t *testing.T) {
	etcdServer, cancel := newEtcd(t)
	defer cancel()

	v3cli := v3client.New(etcdServer)

	dsy, _ := discovery.NewDiscovery(v3cli)
	s, _ := selector.NewSelector(selector.Discovery(dsy))

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/foo/bar", func(w http.ResponseWriter, r *http.Request) {
		// only accept post
		if r.Method != "POST" {
			http.Error(w, "expect post method", 500)
			return
		}

		// hijack the connection
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "could not hijack conn", 500)
			return

		}

		// hijacked
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer conn.Close()

		// read off the first request
		// get codec
		ct := r.Header.Get("Content-Type")
		codec, ok := defaultHTTPCodecs[ct]
		if !ok {
			http.Error(w, "codec not found", 500)
			return
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// extract message
		msg := new(test.Message)
		if err := codec.Unmarshal(b, msg); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// marshal response
		b, err = codec.Marshal(msg)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// write response
		rsp := &http.Response{
			Header:        r.Header,
			Body:          &buffer{bytes.NewBuffer(b)},
			Status:        "200 OK",
			StatusCode:    200,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(b)),
		}

		// write response
		rsp.Write(bufrw)
		bufrw.Flush()

		reader := bufio.NewReader(conn)

		for {
			r, err := http.ReadRequest(reader)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			b, err = io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			// extract message
			msg := new(test.Message)
			if err := codec.Unmarshal(b, msg); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			// marshal response
			b, err = codec.Marshal(msg)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			rsp := &http.Response{
				Header:        r.Header,
				Body:          &buffer{bytes.NewBuffer(b)},
				Status:        "200 OK",
				StatusCode:    200,
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				ContentLength: int64(len(b)),
			}

			// write response
			rsp.Write(bufrw)
			bufrw.Flush()
		}
	})
	go http.Serve(l, mux)

	if err := dsy.Register(context.TODO(), &pb.Service{
		Name: "test.service",
		Nodes: []*pb.Node{
			{
				Id:      "test.service.1",
				Address: l.Addr().String(),
				Metadata: map[string]string{
					"protocol": "http",
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	c, err := NewClient(client.Discovery(dsy), client.Selector(s))
	if err != nil {
		t.Fatal(err)
	}
	req := c.NewRequest("test.service", "/foo/bar", new(test.Message))
	stream, err := c.Stream(context.TODO(), req)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	for i := 0; i < 10; i++ {
		msg := &test.Message{
			Seq:  int64(i),
			Data: fmt.Sprintf("message %d", i),
		}
		err := stream.Send(msg)
		if err != nil {
			t.Fatal(err)
		}
		rsp := new(test.Message)
		err = stream.Recv(rsp)
		if err != nil {
			t.Fatal(err)
		}
		if rsp.Seq != msg.Seq {
			t.Fatalf("invalid seq %d for %d", rsp.Seq, msg.Seq)
		}
	}
}
