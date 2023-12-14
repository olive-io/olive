// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, doftware
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/olive-io/bpmn/flow_node/activity"
	"github.com/olive-io/bpmn/flow_node/activity/call"
	"github.com/olive-io/bpmn/flow_node/activity/receive"
	"github.com/olive-io/bpmn/flow_node/activity/script"
	"github.com/olive-io/bpmn/flow_node/activity/send"
	"github.com/olive-io/bpmn/flow_node/activity/service"
	"github.com/olive-io/bpmn/flow_node/activity/task"
	"github.com/olive-io/bpmn/flow_node/activity/user"
	pb "github.com/olive-io/olive/api/discoverypb"
)

const (
	HeaderKeyPrefix = "ov:"
	NodeIdKey       = "ov:node_id"
	ActivityIdKey   = "ov:activity_id"
)

type DiscoverOptions struct {
	ctx context.Context

	// the kind of BPMN2 Activity
	activity pb.Activity
	// the id of Activity Executor which dependency with Activity Node
	activityId string
	// the id of Activity Node (like olive-gateway, olive-executor)
	nodeId  string
	timeout time.Duration

	task    *TaskOptions
	service *ServiceOptions
	script  *ScriptOptions
	user    *UserOptions
	call    *CallOptions
	send    *SendOptions
	receive *ReceiveOptions
}

func (do *DiscoverOptions) Validate() error {
	if do.ctx == nil {
		do.ctx = context.TODO()
	}

	var err error
	if do.activity == pb.Activity_Unknown {
		return fmt.Errorf("missing field 'activity'")
	}

	act := do.activity
	switch act {
	case pb.Activity_Task:
		if err = do.task.Validate(); err != nil {
			return err
		}
	case pb.Activity_Service:
		if err = do.service.Validate(); err != nil {
			return err
		}
	case pb.Activity_Script:
		if err = do.script.Validate(); err != nil {
			return err
		}
	case pb.Activity_User:
		if err = do.user.Validate(); err != nil {
			return err
		}
	case pb.Activity_Call:
		if err = do.call.Validate(); err != nil {
			return err
		}
	case pb.Activity_Send:
		if err = do.task.Validate(); err != nil {
			return err
		}
	case pb.Activity_Receive:
		if err = do.task.Validate(); err != nil {
			return err
		}
	}

	return err
}

func DiscoverOptionsFromTrace(trace activity.ActiveTaskTrace) []DiscoverOption {
	options := make([]DiscoverOption, 0)

	var actOp DiscoverOption
	var headers map[string]any
	switch tv := trace.(type) {
	case *task.ActiveTrace:
	case *service.ActiveTrace:
		headers = tv.Headers
	case *script.ActiveTrace:
		headers = tv.Headers
	case *user.ActiveTrace:
		headers = tv.Headers
	case *call.ActiveTrace:
	case *send.ActiveTrace:
		headers = tv.Headers
	case *receive.ActiveTrace:
		headers = tv.Headers
	default:
		return options
	}

	options = append(options, actOp)
	if value, ok := headers[NodeIdKey]; ok {
		vv, _ := value.(string)
		options = append(options, DiscoverWithNodeId(vv))
	}
	if value, ok := headers[ActivityIdKey]; ok {
		vv, _ := value.(string)
		options = append(options, DiscoverWithActivityId(vv))
	}

	return options
}

type DiscoverOption func(options *DiscoverOptions)

type TaskOptions struct {
}

func (o *TaskOptions) Validate() (err error) { return }

type ServiceOptions struct {
	// service protocol (gRPC, http, etc.)
	protocol string
	// the Content-Type of request
	contentType string
}

func (o *ServiceOptions) Validate() (err error) { return }

type ScriptOptions struct {
}

func (o *ScriptOptions) Validate() (err error) { return }

type UserOptions struct {
}

func (o *UserOptions) Validate() (err error) { return }

type CallOptions struct {
}

func (o *CallOptions) Validate() (err error) { return }

type SendOptions struct {
}

func (o *SendOptions) Validate() (err error) { return }

type ReceiveOptions struct {
}

func (o *ReceiveOptions) Validate() (err error) { return }

func DiscoverWithActivityId(id string) DiscoverOption {
	return func(do *DiscoverOptions) {
		do.activityId = id
	}
}

func DiscoverWithNodeId(id string) DiscoverOption {
	return func(do *DiscoverOptions) {
		do.nodeId = id
	}
}

func DiscoverWithTimeout(timeout time.Duration) DiscoverOption {
	return func(do *DiscoverOptions) {
		do.timeout = timeout
	}
}

type ExecuteOptions struct {
	timeout time.Duration
}
