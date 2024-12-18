package client

import (
	"context"
	"fmt"

	json "github.com/bytedance/sonic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/olive-io/olive/api"
	pb "github.com/olive-io/olive/api/runnerpb"
	corev1 "github.com/olive-io/olive/api/types/core/v1"
	metav1 "github.com/olive-io/olive/api/types/meta/v1"
)

type Client struct {
	cfg *Config

	scheme *api.Scheme
}

func NewClient(cfg *Config, scheme *api.Scheme) (*Client, error) {

	client := &Client{
		cfg:    cfg,
		scheme: scheme,
	}

	conn, err := client.newConn()
	if err != nil {
		return nil, err
	}
	_ = conn.Close()

	return client, nil
}

func (c *Client) newConn() (*grpc.ClientConn, error) {
	target := c.cfg.Address

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(target, options...)
	return conn, err
}

func (c *Client) getCallOptions() []grpc.CallOption {
	options := []grpc.CallOption{}

	return options
}

func (c *Client) GetRunner(ctx context.Context) (*corev1.Runner, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	in := &pb.GetRunnerRequest{}
	options := c.getCallOptions()

	resp, err := cc.GetRunner(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.Runner, nil
}

func (c *Client) ListDefinitions(ctx context.Context, id string) (*corev1.DefinitionList, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	in := &pb.ListDefinitionsRequest{
		Id: id,
	}
	options := c.getCallOptions()

	resp, err := cc.ListDefinitions(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.List, nil
}

func (c *Client) GetDefinition(ctx context.Context, id string, version int64) (*corev1.Definition, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	in := &pb.GetDefinitionRequest{
		Id:      id,
		Version: version,
	}
	options := c.getCallOptions()

	resp, err := cc.GetDefinition(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.Definition, nil
}

func (c *Client) ListProcessInstances(ctx context.Context, definition string, version int64) (*corev1.ProcessInstanceList, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	in := &pb.ListProcessInstancesRequest{
		DefinitionId:      definition,
		DefinitionVersion: version,
	}
	options := c.getCallOptions()

	resp, err := cc.ListProcessInstances(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.List, nil
}

func (c *Client) GetProcessInstance(ctx context.Context, definition string, version int64, id string) (*corev1.ProcessInstance, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	in := &pb.GetProcessInstanceRequest{
		DefinitionId:      definition,
		DefinitionVersion: version,
		Id:                id,
	}
	options := c.getCallOptions()

	resp, err := cc.GetProcessInstance(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.Instance, nil
}

func (c *Client) RunProcessInstance(ctx context.Context, in *pb.RunProcessInstanceRequest) (*corev1.ProcessInstance, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	options := c.getCallOptions()

	resp, err := cc.RunProcessInstance(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.Instance, nil
}

func (c *Client) List(ctx context.Context, options *metav1.ListOptions) (*metav1.Result, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	opts := c.getCallOptions()

	in := &pb.ListRequest{
		Options: options,
	}
	resp, err := cc.List(ctx, in, opts...)
	if err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func (c *Client) Get(ctx context.Context, options *metav1.GetOptions) (*metav1.Result, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	opts := c.getCallOptions()

	in := &pb.GetRequest{
		Options: options,
	}
	resp, err := cc.Get(ctx, in, opts...)
	if err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func (c *Client) Post(ctx context.Context, options *metav1.PostOptions) error {
	conn, err := c.newConn()
	if err != nil {
		return err
	}

	cc := pb.NewRunnerRPCClient(conn)
	opts := c.getCallOptions()

	in := &pb.PostRequest{
		Options: options,
	}
	_, err = cc.Post(ctx, in, opts...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Patch(ctx context.Context, options *metav1.PatchOptions) error {
	conn, err := c.newConn()
	if err != nil {
		return err
	}

	cc := pb.NewRunnerRPCClient(conn)
	opts := c.getCallOptions()

	in := &pb.PatchRequest{
		Options: options,
	}
	_, err = cc.Patch(ctx, in, opts...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Delete(ctx context.Context, options *metav1.DeleteOptions) error {
	conn, err := c.newConn()
	if err != nil {
		return err
	}

	cc := pb.NewRunnerRPCClient(conn)
	opts := c.getCallOptions()

	in := &pb.DeleteRequest{
		Options: options,
	}
	_, err = cc.Delete(ctx, in, opts...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) BuildProcessInstance() *processBuilder {
	builder := &processBuilder{
		RunProcessInstanceRequest: &pb.RunProcessInstanceRequest{},
		cc:                        c,
	}

	return builder
}

type processBuilder struct {
	*pb.RunProcessInstanceRequest

	cc *Client
}

func (b *processBuilder) SetDefinition(id string, version int64) *processBuilder {
	b.DefinitionId = id
	b.DefinitionVersion = version
	return b
}

func (b *processBuilder) SetBpmn(content string) *processBuilder {
	b.Content = content
	return b
}

func (b *processBuilder) SetName(name string) *processBuilder {
	b.InstanceName = name
	return b
}

func (b *processBuilder) SetHeaders(headers map[string]string) *processBuilder {
	b.Headers = headers
	return b
}

func (b *processBuilder) SetProperties(properties map[string]any) *processBuilder {
	b.Properties = map[string][]byte{}
	for k, v := range properties {
		b.Properties[k] = marshalValue(v)
	}

	return b
}

func (b *processBuilder) SetDataObjects(dataObjects map[string]any) *processBuilder {
	b.DataObjects = map[string][]byte{}
	for k, v := range dataObjects {
		b.DataObjects[k] = marshalValue(v)
	}
	return b
}

func (b *processBuilder) Do(ctx context.Context) (*corev1.ProcessInstance, error) {
	return b.cc.RunProcessInstance(ctx, b.RunProcessInstanceRequest)
}

func marshalValue(v any) []byte {
	switch t := v.(type) {
	case []byte:
		return t
	case string:
		return []byte(t)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return []byte(fmt.Sprintf("%d", t))
	case float32, float64:
		return []byte(fmt.Sprintf("%f", t))
	case bool:
		return []byte(fmt.Sprintf("%t", t))
	default:
		data, _ := json.Marshal(t)
		return data
	}
}
