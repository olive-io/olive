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
	pb "github.com/olive-io/olive/api/discoverypb"
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

func DiscoverOptionsFromTrace(trace *activity.Trace) []DiscoverOption {
	options := make([]DiscoverOption, 0)

	var actOp DiscoverOption
	headers := trace.GetHeaders()
	switch trace.GetActivity().Type() {
	case activity.TaskType:
		actOp = DiscoverWithTask(&TaskOptions{})
	case activity.ServiceType:
		actOp = DiscoverWithService(ServiceOptionsFromHeaders(headers))
	case activity.ScriptType:
		actOp = DiscoverWithScript(ScriptOptionsFromHeaders(headers))
	case activity.UserType:
		actOp = DiscoverWithUser(UserOptionsFromHeaders(headers))
	case activity.CallType:
		actOp = DiscoverWithCall(CallOptionsFromHeaders(headers))
	case activity.SendType:
		actOp = DiscoverWithSend(SendOptionsFromHeaders(headers))
	case activity.ReceiveType:
		actOp = DiscoverWithReceive(ReceiveOptionsFromHeaders(headers))
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

func DiscoverWithTask(taskOpt *TaskOptions) DiscoverOption {
	return func(options *DiscoverOptions) {
		options.activity = pb.Activity_Task
		options.task = taskOpt
	}
}

type TaskOptions struct {
}

func (o *TaskOptions) Validate() (err error) { return }

func DiscoverWithService(serviceOpt *ServiceOptions) DiscoverOption {
	return func(options *DiscoverOptions) {
		options.activity = pb.Activity_Service
		options.service = serviceOpt
	}
}

type ServiceOptions struct {
	// service protocol (gRPC, http, etc.)
	protocol string
	// the Content-Type of request
	contentType string
}

func ServiceOptionsFromHeaders(headers map[string]any) *ServiceOptions {
	so := &ServiceOptions{}
	return so
}

func (o *ServiceOptions) Validate() (err error) { return }

func DiscoverWithScript(scriptOpt *ScriptOptions) DiscoverOption {
	return func(options *DiscoverOptions) {
		options.activity = pb.Activity_Script
		options.script = scriptOpt
	}
}

type ScriptOptions struct {
}

func ScriptOptionsFromHeaders(headers map[string]any) *ScriptOptions {
	so := &ScriptOptions{}
	return so
}

func (o *ScriptOptions) Validate() (err error) { return }

func DiscoverWithUser(userOpt *UserOptions) DiscoverOption {
	return func(options *DiscoverOptions) {
		options.activity = pb.Activity_User
		options.user = userOpt
	}
}

type UserOptions struct {
}

func UserOptionsFromHeaders(headers map[string]any) *UserOptions {
	uo := &UserOptions{}
	return uo
}

func (o *UserOptions) Validate() (err error) { return }

func DiscoverWithCall(callOpt *CallOptions) DiscoverOption {
	return func(options *DiscoverOptions) {
		options.activity = pb.Activity_Call
		options.call = callOpt
	}
}

type CallOptions struct {
}

func CallOptionsFromHeaders(headers map[string]any) *CallOptions {
	co := &CallOptions{}
	return co
}

func (o *CallOptions) Validate() (err error) { return }

func DiscoverWithSend(sendOpt *SendOptions) DiscoverOption {
	return func(options *DiscoverOptions) {
		options.activity = pb.Activity_Send
		options.send = sendOpt
	}
}

type SendOptions struct {
}

func SendOptionsFromHeaders(headers map[string]any) *SendOptions {
	so := &SendOptions{}
	return so
}

func (o *SendOptions) Validate() (err error) { return }

func DiscoverWithReceive(receiveOpt *ReceiveOptions) DiscoverOption {
	return func(options *DiscoverOptions) {
		options.activity = pb.Activity_Receive
		options.receive = receiveOpt
	}
}

type ReceiveOptions struct {
}

func ReceiveOptionsFromHeaders(headers map[string]any) *ReceiveOptions {
	ro := &ReceiveOptions{}
	return ro
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

type ExecuteOption func(*ExecuteOptions)

func ExecuteWithTimeout(timeout time.Duration) ExecuteOption {
	return func(options *ExecuteOptions) {
		options.timeout = timeout
	}
}
