/*
Copyright 2025 The olive Authors

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

package clientgo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "github.com/olive-io/olive/api/rpc/serverpb"
	"github.com/olive-io/olive/api/types"
)

type ListDefinitionsOptions struct {
	Page, Size int32
}

type ExecuteProcessOptions struct {
	Name               string
	DefinitionsId      int64
	DefinitionsVersion uint64
	Priority           int64
	Headers            map[string]string
	Properties         map[string]string
	DataObjects        map[string]string
}

type Client struct {
	cfg *Config

	conn *grpc.ClientConn

	bpmnClient   pb.BpmnRPCClient
	systemClient pb.SystemRPCClient

	done chan struct{}
}

func New(cfg *Config) (*Client, error) {
	target := cfg.Target

	var tlsConfig *tls.Config
	if cfgTLS := cfg.TLS; cfgTLS != nil {
		cert, err := tls.LoadX509KeyPair(cfgTLS.CertFile, cfgTLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load certificate pair: %w", err)
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
			MaxVersion:   tls.VersionTLS13,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			},
		}

		if cfgTLS.CaFile != "" {
			caCert, err := os.ReadFile(cfgTLS.CaFile)
			if err != nil {
				return nil, fmt.Errorf("load certificate CA: %w", err)
			}
			caPool := x509.NewCertPool()
			caPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caPool
		}
	}

	var creds credentials.TransportCredentials
	if tlsConfig != nil {
		creds = credentials.NewTLS(tlsConfig)
	} else {
		creds = insecure.NewCredentials()
	}

	kecp := keepalive.ClientParameters{
		Time:                time.Second * 10,
		Timeout:             time.Second * 30,
		PermitWithoutStream: true,
	}

	DialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(kecp),
		grpc.WithIdleTimeout(DefaultTimeout),
	}

	conn, err := grpc.NewClient(target, DialOpts...)
	if err != nil {
		return nil, fmt.Errorf("initialize grpc client: %w", err)
	}

	bpmnClient := pb.NewBpmnRPCClient(conn)
	systemClient := pb.NewSystemRPCClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	callOpts := []grpc.CallOption{}
	_, err = systemClient.Ping(ctx, &pb.PingRequest{}, callOpts...)
	if err != nil {
		return nil, parseErr(err)
	}

	client := &Client{
		cfg:          cfg,
		conn:         conn,
		bpmnClient:   bpmnClient,
		systemClient: systemClient,
		done:         make(chan struct{}, 1),
	}

	return client, nil
}

func (c *Client) GetConn() *grpc.ClientConn {
	return c.conn
}

func (c *Client) Ping(ctx context.Context) error {
	opts := c.buildCallOptions()

	in := &pb.PingRequest{}
	_, err := c.systemClient.Ping(ctx, in, opts...)
	if err != nil {
		return parseErr(err)
	}
	return nil
}

func (c *Client) Register(ctx context.Context, runner *types.Runner, endpoints []*types.Endpoint) (*types.Runner, error) {
	opts := c.buildCallOptions()

	in := &pb.RegisterRequest{
		Runner:    runner,
		Endpoints: endpoints,
	}
	rsp, err := c.systemClient.Register(ctx, in, opts...)
	if err != nil {
		return nil, parseErr(err)
	}
	return rsp.Runner, nil
}

func (c *Client) Disregister(ctx context.Context, uid string) (*types.Runner, error) {
	opts := c.buildCallOptions()

	in := &pb.DisregisterRequest{
		Id: uid,
	}
	rsp, err := c.systemClient.Disregister(ctx, in, opts...)
	if err != nil {
		return nil, parseErr(err)
	}
	return rsp.Runner, nil
}

func (c *Client) DeployDefinitions(ctx context.Context, definitionsXML []byte, desc string, metadata map[string]string) (*types.Definitions, error) {
	req := &pb.DeployDefinitionsRequest{
		Metadata:    metadata,
		Content:     definitionsXML,
		Description: desc,
	}

	opts := c.buildCallOptions()

	rsp, err := c.bpmnClient.DeployDefinition(ctx, req, opts...)
	if err != nil {
		return nil, parseErr(err)
	}
	return rsp.Definitions, nil
}

func (c *Client) ListDefinitions(ctx context.Context, options *ListDefinitionsOptions) ([]*types.Definitions, int64, error) {
	req := &pb.ListDefinitionsRequest{
		Page: options.Page,
		Size: options.Size,
	}

	opts := c.buildCallOptions()
	rsp, err := c.bpmnClient.ListDefinitions(ctx, req, opts...)
	if err != nil {
		return nil, 0, parseErr(err)
	}
	return rsp.DefinitionsList, rsp.Total, nil
}

func (c *Client) GetDefinitions(ctx context.Context, uid string, version uint64) (*types.Definitions, error) {
	req := &pb.GetDefinitionsRequest{
		Uid:     uid,
		Version: version,
	}

	opts := c.buildCallOptions()
	rsp, err := c.bpmnClient.GetDefinitions(ctx, req, opts...)
	if err != nil {
		return nil, parseErr(err)
	}
	return rsp.Definitions, nil
}

func (c *Client) ExecuteProcess(ctx context.Context, options *ExecuteProcessOptions) (*types.Process, error) {
	req := &pb.ExecuteProcessRequest{
		Name:               options.Name,
		DefinitionsId:      options.DefinitionsId,
		DefinitionsVersion: options.DefinitionsVersion,
		Priority:           options.Priority,
		Headers:            options.Headers,
		Properties:         options.Properties,
		DataObjects:        options.DataObjects,
	}
	opts := c.buildCallOptions()
	rsp, err := c.bpmnClient.ExecuteProcess(ctx, req, opts...)
	if err != nil {
		return nil, parseErr(err)
	}
	return rsp.Process, nil
}

func (c *Client) NewConnection(ctx context.Context) (pb.SystemRPC_RunnerConnectionClient, error) {
	opts := c.buildCallOptions()
	stream, err := c.systemClient.RunnerConnection(ctx, opts...)
	if err != nil {
		return nil, parseErr(err)
	}
	return stream, nil
}

func (c *Client) Close() error {
	select {
	case <-c.done:
	default:
		close(c.done)
	}

	return c.conn.Close()
}

func (c *Client) buildCallOptions() []grpc.CallOption {
	opts := []grpc.CallOption{}
	return opts
}
