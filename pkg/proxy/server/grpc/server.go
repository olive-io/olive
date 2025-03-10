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

package grpc

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/cockroachdb/errors"
	"go.uber.org/zap"

	"github.com/olive-io/olive/pkg/proxy/server"
)

var (
	// Precompute the reflection type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()
)

type methodType struct {
	method      reflect.Method
	ArgType     reflect.Type
	ReplyType   reflect.Type
	ContextType reflect.Type
	NumIn       int
	NumOut      int
	stream      bool
}

type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

// server represents an RPC Server.
type rServer struct {
	mu         sync.Mutex // protects the serviceMap
	lg         *zap.Logger
	serviceMap map[string]*service
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// prepareEndpoint() returns a methodType for the provided method or nil
// in case if the method was unsuitable.
func prepareEndpoint(lg *zap.Logger, method reflect.Method) *methodType {
	if lg == nil {
		lg = zap.NewNop()
	}
	log := lg.Sugar()

	mtype := method.Type
	mname := method.Name
	var replyType, argType, contextType reflect.Type
	var stream bool

	// Endpoint() must be exported.
	if method.PkgPath != "" {
		return nil
	}

	in, out := mtype.NumIn(), mtype.NumOut()
	if in == 3 && out == 2 {
		contextType = mtype.In(1)
		argType = mtype.In(2)
		replyType = mtype.Out(0)
	} else if in == 3 && out == 1 {
		argType = mtype.In(1)
		replyType = mtype.In(2)
		stream = true
	} else if in == 2 && out == 1 {
		argType = mtype.In(1)
		replyType = mtype.In(1)
		stream = true
	} else {
		log.Errorf("method %v of %v has wrong number of ins: %v", mname, mtype, mtype.NumIn())
		return nil
	}

	if stream {
		// check stream type
		streamType := reflect.TypeOf((*server.IStream)(nil)).Elem()
		if !argType.Implements(streamType) {
			log.Errorf("%v argument does not implement Streamer interface: %v", mname, argType)
			return nil
		}
	} else {
		// if not stream check the replyType

		// First arg need not be a pointer.
		if !isExportedOrBuiltinType(argType) {
			log.Errorf("%v argument type not exported: %v", mname, argType)
			return nil
		}

		if replyType.Kind() != reflect.Ptr {
			log.Errorf("method %v reply type not a pointer: %v", mname, replyType)
			return nil
		}

		// Reply type must of exported.
		if !isExportedOrBuiltinType(replyType) {
			log.Errorf("method %v reply type not exported: %v", mname, replyType)
			return nil
		}
	}

	return &methodType{
		method:      method,
		ArgType:     argType,
		ReplyType:   replyType,
		ContextType: contextType,
		NumIn:       in,
		NumOut:      out,
		stream:      stream,
	}
}

func (server *rServer) register(rcvr interface{}) error {
	log := server.lg.Sugar()

	server.mu.Lock()
	defer server.mu.Unlock()
	if server.serviceMap == nil {
		server.serviceMap = make(map[string]*service)
	}
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := strings.TrimSuffix(reflect.Indirect(s.rcvr).Type().Name(), "EmbedXX")
	if sname == "" {
		log.Fatalf("rpc: no service name for type %v", s.typ.String())
	}
	if !isExported(sname) {
		s := "rpc Register: type " + sname + " is not exported"
		log.Error(s)
		return errors.New(s)
	}
	if _, present := server.serviceMap[sname]; present {
		return errors.New("rpc: service already defined: " + sname)
	}
	s.name = sname
	s.method = make(map[string]*methodType)

	// Install the methods
	for m := 0; m < s.typ.NumMethod(); m++ {
		method := s.typ.Method(m)
		if mt := prepareEndpoint(server.lg, method); mt != nil {
			s.method[method.Name] = mt
		}
	}

	if len(s.method) == 0 {
		s := "rpc Register: type " + sname + " has no exported methods of suitable type"
		log.Error(s)
		return errors.New(s)
	}
	server.serviceMap[s.name] = s
	return nil
}

func (m *methodType) prepareContext(ctx context.Context) reflect.Value {
	if contextv := reflect.ValueOf(ctx); contextv.IsValid() {
		return contextv
	}
	return reflect.Zero(m.ContextType)
}
