package profile

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var errUnknown = errors.New("unknown error")

func receiveFileChunk(writer io.Writer, stream interface{ Recv() (*FileChunk, error) }) error {
	for {
		fc, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		_, err = writer.Write(fc.Content)
		if err != nil {
			return err
		}
	}
	return nil
}

// Client will start the grpc-profile client
type Client struct {
	client          ProfileServiceClient
	conn            *grpc.ClientConn
	ctx             context.Context
	grpcCallOptions []grpc.CallOption
	grpcDialOptions []grpc.DialOption
}

// SetGRPCCallOption will set new GRPC Call option to GRPC Profile Client
func (client *Client) SetGRPCCallOption(grpcCallOption grpc.CallOption) {
	client.grpcCallOptions = append(client.grpcCallOptions, grpcCallOption)
}

// SetGRPCDialOption will set new GRPC Dial option to GRPC Profile Client
func (client *Client) SetGRPCDialOption(grpcDialOption grpc.DialOption) {
	client.grpcDialOptions = append(client.grpcDialOptions, grpcDialOption)
}

// AuthTypeDialInsecure will set insecure auth type to grpc client
func AuthTypeDialInsecure() struct {
	DialOption grpc.DialOption
	Error      error
} {
	return struct {
		DialOption grpc.DialOption
		Error      error
	}{DialOption: grpc.WithInsecure()}
}

// AuthTypeDialTLS will set TLS auth type to grpc client
func AuthTypeDialTLS(certFile string) struct {
	DialOption grpc.DialOption
	Error      error
} {
	cred, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		return struct {
			DialOption grpc.DialOption
			Error      error
		}{Error: err}
	}
	return struct {
		DialOption grpc.DialOption
		Error      error
	}{DialOption: grpc.WithTransportCredentials(cred)}
}

// Connect client to GRPC Profile Server
func (client *Client) Connect(ctx context.Context, serverAddress string) error {
	conn, err := grpc.Dial(serverAddress, client.grpcDialOptions...)
	if err != nil {
		return err
	}
	client.conn = conn
	client.client = NewProfileServiceClient(client.conn)
	client.ctx = ctx
	return nil
}

// NewClient function will create a new GRPC Profile Client instance
func NewClient(ctx context.Context, serverAddress string, authType struct {
	DialOption grpc.DialOption
	Error      error
}, grpcDialOptions ...grpc.DialOption) (*Client, error) {
	client := Client{}

	// Security
	if authType.Error != nil {
		return nil, authType.Error
	}
	client.SetGRPCDialOption(authType.DialOption)

	// Other dial options
	for _, dialOption := range grpcDialOptions {
		client.SetGRPCDialOption(dialOption)
	}
	err := client.Connect(ctx, serverAddress)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// Stop function will stop GRPC Profile Client instance
func (client *Client) Stop() error {
	return client.conn.Close()
}

// ClearProfileCache will clear cached profiles in server
func (client *Client) ClearProfileCache(grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.ClearProfileCache(client.ctx, &empty.Empty{}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// SetMemProfileRate will set memory profile rate in server
func (client *Client) SetMemProfileRate(rate int, grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.SetMemProfileRate(client.ctx, &Rate{Value: int64(rate)}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// SetMutexProfileFraction will set mutex profile fraction in server
func (client *Client) SetMutexProfileFraction(rate int, grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.SetMutexProfileFraction(client.ctx, &Rate{Value: int64(rate)}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// SetBlockProfileRate will set block profile rate in server
func (client *Client) SetBlockProfileRate(rate int, grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.SetBlockProfileRate(client.ctx, &Rate{Value: int64(rate)}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// ResetMemProfileRate will set the memory profile rate to default value in server
func (client *Client) ResetMemProfileRate(grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.ResetMemProfileRate(client.ctx, &empty.Empty{}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// ResetMutexProfileFraction will set the mutex profile fraction to default value in server
func (client *Client) ResetMutexProfileFraction(grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.ResetMutexProfileFraction(client.ctx, &empty.Empty{}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// ResetBlockProfileRate will set the block profile rate to default value in server
func (client *Client) ResetBlockProfileRate(grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.ResetBlockProfileRate(client.ctx, &empty.Empty{}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// CPU function will collect CPU profile in server
func (client *Client) CPU(duration time.Duration, grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.CPU(client.ctx, ptypes.DurationProto(duration), append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// Memory function will collect Memory profile in server
func (client *Client) Memory(grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.Memory(client.ctx, &empty.Empty{}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// Mutex function will collect Mutex profile in server
func (client *Client) Mutex(grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.Mutex(client.ctx, &empty.Empty{}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// Block function will collect Block profile in server
func (client *Client) Block(grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.Block(client.ctx, &empty.Empty{}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// ThreadCreate function will collect ThreadCreate profile in server
func (client *Client) ThreadCreate(grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.ThreadCreate(client.ctx, &empty.Empty{}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// GoRoutine function will collect GoRoutine profile in server
func (client *Client) GoRoutine(grpcCallOption ...grpc.CallOption) error {
	s, err := client.client.GoRoutine(client.ctx, &empty.Empty{}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	if s.Code != StatusCode_OK {
		return errUnknown
	}
	return nil
}

// Trace function will collect trace data in server and download it
func (client *Client) Trace(duration time.Duration, writer io.Writer, grpcCallOption ...grpc.CallOption) error {
	downloadClient, err := client.client.Trace(client.ctx, ptypes.DurationProto(duration), append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	err = receiveFileChunk(writer, downloadClient)
	if err != nil {
		return err
	}
	return nil
}

// Download function will download the collected profile data from server
func (client *Client) Download(writer io.Writer, grpcCallOption ...grpc.CallOption) error {
	downloadClient, err := client.client.Download(client.ctx, &empty.Empty{}, append(client.grpcCallOptions, grpcCallOption...)...)
	if err != nil {
		return err
	}
	err = receiveFileChunk(writer, downloadClient)
	if err != nil {
		return err
	}
	return nil
}
