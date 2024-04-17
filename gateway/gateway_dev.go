//go:build dev

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

package gateway

import (
	"context"
	"time"

	"go.uber.org/zap"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/gatewaypb"
	"github.com/olive-io/olive/gateway/consumer"
)

func (g *Gateway) installHandler() (err error) {
	if err = pb.RegisterTestServiceServerHandler(g, &TestRPC{}); err != nil {
		return err
	}

	if err = g.installInternalHandler(); err != nil {
		return
	}

	if err = g.AddConsumer(&ScriptTaskConsumer{},
		AddWithRequest(&ScriptTaskRequest{}),
		AddWithResponse(&ScriptTaskResponse{}),
		AddWithActivity(dsypb.ActivityType_ScriptTask),
		AddWithAction("tengo"),
	); err != nil {
		return err
	}
	if err = g.AddConsumer(&SendTaskConsumer{},
		AddWithRequest(&SendTaskRequest{}),
		AddWithResponse(&SendTaskResponse{}),
		AddWithActivity(dsypb.ActivityType_SendTask),
		AddWithAction("rabbitmq"),
	); err != nil {
		return err
	}

	g.AddHandlerWrapper(TestHandlerWrapper(g.Logger()))

	return nil
}

type TestRPC struct {
	pb.UnsafeTestServiceServer
}

func (t *TestRPC) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Reply: "hello world"}, nil
}

func TestHandlerWrapper(lg *zap.Logger) consumer.HandlerWrapper {
	return func(h consumer.HandlerFunc) consumer.HandlerFunc {
		return func(ctx *consumer.Context) (any, error) {
			start := time.Now()
			rsp, err := h(ctx)
			speed := time.Now().Sub(start)
			lg.Info("handle task", zap.Stringer("speed", speed))
			return rsp, err
		}
	}
}

type ScriptTaskRequest struct {
	Expr string `json:"expr"`
}

type ScriptTaskResponse struct {
	Sum int64 `json:"sum"`
}

type ScriptTaskConsumer struct{}

func (c *ScriptTaskConsumer) Handle(ctx *consumer.Context) (any, error) {
	return 1, nil
}

type SendTaskRequest struct {
	Topic    string `json:"topic"`
	Username string `json:"username"`
}

type SendTaskResponse struct {
	Result string `json:"result"`
}

type SendTaskConsumer struct{}

func (c *SendTaskConsumer) Handle(ctx *consumer.Context) (any, error) {
	return &SendTaskResponse{Result: "OK"}, nil
}
