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

package brain

import (
	"context"
	"net"

	dragonboat "github.com/lni/dragonboat/v4"
	"github.com/oliveio/olive/api"
	"google.golang.org/grpc"
)

func Bootstrap(ctx context.Context, option Option) error {

	nh, err := dragonboat.NewNodeHost(option.NodeHost)
	if err != nil {
		return err
	}

	join := false
	if option.InitialClusterState == ExistingCluster {
		join = true
	}
	initialMembers := option.discoveryMembers()

	if err = nh.StartOnDiskReplica(initialMembers, join, NewDiskKV, option.Node); err != nil {
		return err
	}
	//nh.SyncPropose()

	s := &Server{
		Option: option,
		ctx:    ctx,
		nh:     nh,
		done:   make(chan struct{}, 1),
		exit:   make(chan struct{}, 1),
	}

	listener, err := net.Listen("tcp", s.ListenAddress)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	api.RegisterDefinitionRPCServer(grpcServer, s)

	if err = grpcServer.Serve(listener); err != nil {
		return err
	}

	return nil
}
