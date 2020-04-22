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

// Variable is type for GRPC Profile Variable
type Variable int

// LookupType is type for GRPC Profile LookupType
type LookupType int

// NonLookupType is type for GRPC Profile NonLookupType
type NonLookupType int

const (
	// MemProfRate controls the fraction of memory allocations that are recorded and reported in the memory profile.
	// The profiler aims to sample an average of one allocation per MemProfileRate bytes allocated.
	// To include every allocated block in the profile, set MemProfileRate to 1. To turn off profiling entirely, set
	// MemProfileRate to 0.
	MemProfRate Variable = iota

	// MutexProfileFraction controls the fraction of mutex contention events that are reported in the mutex profile.
	// On average 1/rate events are reported. To turn off profiling entirely, pass rate 0.
	MutexProfileFraction

	// BlockProfileRate controls the fraction of goroutine blocking events that are reported in the blocking profile.
	// The profiler aims to sample an average of one blocking event per rate nanoseconds spent blocked.
	// To include every blocking event in the profile, pass rate = 1. To turn off profiling entirely, pass rate <= 0.
	BlockProfileRate
)
const (
	// HeapType - Memory / Heap Profile Type
	HeapType LookupType = iota
	// MutexType - Mutex Profile Type
	MutexType
	// BlockType - Block Profile Type
	BlockType
	// ThreadCreateType - ThreadCreate Profile Type
	ThreadCreateType
	// GoRoutineType - GoRoutine Profile Type
	GoRoutineType
)
const (
	// CPUType - CPU profile type
	CPUType NonLookupType = iota

	// TraceType - Trace Profile Type
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

// Client will store GRPC Profile Client instance. We can create a instance of the client using `NewClient()` function
type Client struct {
	client      proto.ProfileServiceClient
	conn        *grpc.ClientConn
	ctx         context.Context
	callOptions []grpc.CallOption
	dialOptions []grpc.DialOption
}

// DialOption will create a Dial Option for the GRPC Profile Client
type DialOption struct {
	option grpc.DialOption
	error  error
}

// CallOption will create a Call Option for the GRPC Profile Client
type CallOption struct {
	option grpc.CallOption
	error  error
}

// SetDialOption function will be used to set `DialOption` to GRPC Profile Client
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

// SetDialOptions function will be used to set `DialOption`s to GRPC Profile Client
func (client *Client) SetDialOptions(options ...*DialOption) (err error) {
	for _, option := range options {
		err = client.SetDialOption(option)
		if err != nil {
			return
		}
	}
	return
}

// SetCallOption function will be used to set `CallOption` to GRPC Profile Client
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

// SetCallOptions function will be used to set `CallOption`s to GRPC Profile Client
func (client *Client) SetCallOptions(options ...*CallOption) (err error) {
	for _, option := range options {
		err = client.SetCallOption(option)
		if err != nil {
			return
		}
	}
	return
}

// DialAuthTypeInsecure function will create a Insecure Auth type GRPC Profile Client Dial option
func DialAuthTypeInsecure() *DialOption {
	return &DialOption{option: grpc.WithInsecure()}
}

// DialAuthTypeTLS function will create a TLS Secure Auth type GRPC Profile Client Dial option
func DialAuthTypeTLS(certFile string) *DialOption {
	cred, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		return &DialOption{error: err}
	}
	return &DialOption{option: grpc.WithTransportCredentials(cred)}
}

// NewClient function will create a GRPC Profile Client instance
func NewClient(ctx context.Context, serverAddress string, options ...*DialOption) (client *Client, err error) {
	_ = client.SetDialOption(DialAuthTypeInsecure()) // Default insecure security

	err = client.SetDialOptions(options...)
	if err != nil {
		return
	}

	err = client.Connect(ctx, serverAddress)
	return
}

// Connect function will connect GRPC Profile Client to GRPC Profile Server
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

// Stop function will stop GRPC Profile Client
func (client *Client) Stop() error {
	return client.conn.Close()
}

// ClearProfileCache function will clear all saved profiles in the GRPC Profile Server
func (client *Client) ClearProfileCache(ctx context.Context) (err error) {
	_, err = client.client.ClearProfileCache(ctx, &emptypb.Empty{}, client.callOptions...)
	return
}

// Set function will set the GRPC Profile Variable
func (client *Client) Set(ctx context.Context, v Variable, r int) (err error) {
	_, err = client.client.Set(ctx, &proto.SetProfileInputType{Variable: lookupVariable[v], Rate: int32(r)}, client.callOptions...)
	return
}

// Reset function will reset the GRPC Profile Variable to its original value
func (client *Client) Reset(ctx context.Context, v Variable) (err error) {
	_, err = client.client.Reset(ctx, &proto.ResetProfileInputType{Variable: lookupVariable[v]}, client.callOptions...)
	return
}

// LookupProfile will run a profile for lookup pprof type
func (client *Client) LookupProfile(ctx context.Context, t LookupType, writer io.Writer, keep bool) error {
	stream, err := client.client.LookupProfile(ctx, &proto.LookupProfileInputType{ProfileType: lookupLookupType[t], Keep: keep}, client.callOptions...)
	if err != nil {
		return err
	}
	return receiveFileChunk(writer, stream)
}

// DownloadLookupProfile will download a lookup profile type storred in GRPC Profile Server
func (client *Client) DownloadLookupProfile(ctx context.Context, t LookupType, writer io.Writer) error {
	stream, err := client.client.DownloadLookupProfile(ctx, &proto.LookupProfileType{Profile: lookupLookupType[t]}, client.callOptions...)
	if err != nil {
		return err
	}
	return receiveFileChunk(writer, stream)
}

// NonLookupProfile will run a profile for non lookup pprof type
func (client *Client) NonLookupProfile(ctx context.Context, t NonLookupType, d time.Duration, writer io.Writer, wait, keep bool) error {
	stream, err := client.client.NonLookupProfile(ctx, &proto.NonLookupProfileInputType{ProfileType: lookupNonLookupType[t], Duration: ptypes.DurationProto(d), WaitForCompletion: wait, Keep: keep}, client.callOptions...)
	if err != nil {
		return err
	}
	return receiveFileChunk(writer, stream)
}

// StopNonLookupProfile will stop non lookup profile type (if running)
func (client *Client) StopNonLookupProfile(ctx context.Context, t NonLookupType) (err error) {
	_, err = client.client.StopNonLookupProfile(ctx, &proto.NonLookupProfileType{Profile: lookupNonLookupType[t]}, client.callOptions...)
	return
}

// DownloadNonLookupProfile will download a non lookup profile type storred in GRPC Profile Server
func (client *Client) DownloadNonLookupProfile(ctx context.Context, t NonLookupType, writer io.Writer) error {
	stream, err := client.client.DownloadNonLookupProfile(ctx, &proto.NonLookupProfileType{Profile: lookupNonLookupType[t]}, client.callOptions...)
	if err != nil {
		return err
	}
	return receiveFileChunk(writer, stream)
}
