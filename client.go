package profile

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/chanchal1987/grpc-profile/proto"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"
)

func receiveFileChunk(writer io.Writer, stream interface {
	Recv() (*proto.FileChunk, error)
}) (err error) {
	var fc *proto.FileChunk

	for {
		fc, err = stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return
			}
		}
		_, err = writer.Write(fc.Content)
		if err != nil {
			return
		}
	}
	return
}

type Variable int
type LookupType int
type NonLookupType int

const (
	MemProfRate Variable = iota
	MutexProfileFraction
	BlockProfileRate
)
const (
	HeapType LookupType = iota
	MutexType
	BlockType
	ThreadCreateType
	GoRoutineType
)
const (
	CPUType NonLookupType = iota
	TraceType
)

var lookupVariable = map[Variable]proto.ProfileVariable{
	MemProfRate:          proto.ProfileVariable_MemProfileRate,
	MutexProfileFraction: proto.ProfileVariable_MutexProfileFraction,
	BlockProfileRate:     proto.ProfileVariable_BlockProfileRate,
}
var lookupLookupType = map[LookupType]proto.LookupProfile{
	HeapType:         proto.LookupProfile_profileTypeHeap,
	MutexType:        proto.LookupProfile_profileTypeMutex,
	BlockType:        proto.LookupProfile_profileTypeBlock,
	ThreadCreateType: proto.LookupProfile_profileTypeThreadCreate,
	GoRoutineType:    proto.LookupProfile_profileTypeGoRoutine,
}
var lookupNonLookupType = map[NonLookupType]proto.NonLookupProfile{
	CPUType:   proto.NonLookupProfile_profileTypeCPU,
	TraceType: proto.NonLookupProfile_profileTypeTrace,
}

type Client struct {
	client      proto.ProfileServiceClient
	conn        *grpc.ClientConn
	ctx         context.Context
	callOptions []grpc.CallOption
	dialOptions []grpc.DialOption
}

type DialOption struct {
	option grpc.DialOption
	error  error
}

type CallOption struct {
	option grpc.CallOption
	error  error
}

func (client *Client) SetDialOption(option *DialOption) error {
	if option == nil {
		return nil
	}
	if option.error != nil {
		return option.error
	}
	client.dialOptions = append(client.dialOptions, option.option)
	return nil
}

func (client *Client) SetDialOptions(options ...*DialOption) (err error) {
	for _, option := range options {
		err = client.SetDialOption(option)
		if err != nil {
			return
		}
	}
	return
}

func (client *Client) SetCallOption(option *CallOption) error {
	if option == nil {
		return nil
	}
	if option.error != nil {
		return option.error
	}
	client.callOptions = append(client.callOptions, option.option)
	return nil
}

func (client *Client) SetCallOptions(options ...*CallOption) (err error) {
	for _, option := range options {
		err = client.SetCallOption(option)
		if err != nil {
			return
		}
	}
	return
}

func DialAuthTypeInsecure() *DialOption {
	return &DialOption{option: grpc.WithInsecure()}
}

func DialAuthTypeTLS(certFile string) *DialOption {
	cred, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		return &DialOption{error: err}
	}
	return &DialOption{option: grpc.WithTransportCredentials(cred)}
}

func NewClient(ctx context.Context, serverAddress string, options ...*DialOption) (client *Client, err error) {
	_ = client.SetDialOption(DialAuthTypeInsecure()) // Default insecure security

	err = client.SetDialOptions(options...)
	if err != nil {
		return
	}

	err = client.Connect(ctx, serverAddress)
	return
}

func (client *Client) Connect(ctx context.Context, serverAddress string) error {
	conn, err := grpc.Dial(serverAddress, client.dialOptions...)
	if err != nil {
		return err
	}
	client.ctx = ctx
	client.conn = conn
	client.client = proto.NewProfileServiceClient(client.conn)

	repl, err := client.client.Ping(ctx, &emptypb.Empty{}, client.callOptions...)
	if err != nil {
		return err
	}
	if repl.Message != "pong" {
		return errors.New("unknown error")
	}
	return nil
}

func (client *Client) Stop() error {
	return client.conn.Close()
}

func (client *Client) ClearProfileCache(ctx context.Context) (err error) {
	_, err = client.client.ClearProfileCache(ctx, &emptypb.Empty{}, client.callOptions...)
	return
}

func (client *Client) Set(ctx context.Context, v Variable, r int) (err error) {
	_, err = client.client.Set(ctx, &proto.SetProfileInputType{Variable: lookupVariable[v], Rate: int32(r)}, client.callOptions...)
	return
}

func (client *Client) Reset(ctx context.Context, v Variable) (err error) {
	_, err = client.client.Reset(ctx, &proto.ResetProfileInputType{Variable: lookupVariable[v]}, client.callOptions...)
	return
}

func (client *Client) LookupProfile(ctx context.Context, t LookupType, writer io.Writer, keep bool) error {
	stream, err := client.client.LookupProfile(ctx, &proto.LookupProfileInputType{ProfileType: lookupLookupType[t], Keep: keep}, client.callOptions...)
	if err != nil {
		return err
	}
	return receiveFileChunk(writer, stream)
}

func (client *Client) DownloadLookupProfile(ctx context.Context, t LookupType, writer io.Writer) error {
	stream, err := client.client.DownloadLookupProfile(ctx, &proto.LookupProfileType{Profile: lookupLookupType[t]}, client.callOptions...)
	if err != nil {
		return err
	}
	return receiveFileChunk(writer, stream)
}

func (client *Client) NonLookupProfile(ctx context.Context, t NonLookupType, d time.Duration, writer io.Writer, wait, keep bool) error {
	stream, err := client.client.NonLookupProfile(ctx, &proto.NonLookupProfileInputType{ProfileType: lookupNonLookupType[t], Duration: ptypes.DurationProto(d), WaitForCompletion: wait, Keep: keep}, client.callOptions...)
	if err != nil {
		return err
	}
	return receiveFileChunk(writer, stream)
}

func (client *Client) StopNonLookupProfile(ctx context.Context, t NonLookupType) (err error) {
	_, err = client.client.StopNonLookupProfile(ctx, &proto.NonLookupProfileType{Profile: lookupNonLookupType[t]}, client.callOptions...)
	return
}

func (client *Client) DownloadNonLookupProfile(ctx context.Context, t NonLookupType, writer io.Writer) error {
	stream, err := client.client.DownloadNonLookupProfile(ctx, &proto.NonLookupProfileType{Profile: lookupNonLookupType[t]}, client.callOptions...)
	if err != nil {
		return err
	}
	return receiveFileChunk(writer, stream)
}
