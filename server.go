package profile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/pprof/profile"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func equalValueType(st1, st2 *profile.ValueType) bool {
	return st1.Type == st2.Type && st1.Unit == st2.Unit
}

func sendFileChunk(reader io.Reader, stream interface{ Send(*FileChunk) error }) error {
	for {
		var b byte
		_, err := reader.Read([]byte{b})
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		err = stream.Send(&FileChunk{Content: []byte{b}})
		if err != nil {
			return err
		}
	}
	return nil
}

// Server will start the grpc-profile server. This has to be called in the application where we want to run profiles.
type Server struct {
	Quite                   bool
	Logger                  io.Writer
	profiles                []*profile.Profile
	listen                  net.Listener
	server                  *grpc.Server
	oldMemProfileRate       int
	oldMutexProfileFraction int
	grpcServerOptions       []grpc.ServerOption
}

func (server *Server) log(format string, args ...interface{}) error {
	if server.Quite {
		return nil
	}
	_, err := fmt.Fprintf(server.Logger, "Profile: "+format+"\n", args...)
	return err
}

func (server *Server) err(text string) error {
	return errors.New("GRPC Profile Server Error: " + text)
}

func (server Server) collectLookupProfile(ctx context.Context, lookupName string) error {
	if prof := pprof.Lookup(lookupName); prof != nil {
		var buf bytes.Buffer
		err := prof.WriteTo(&buf, 0)
		if err != nil {
			return err
		}
		p, err := profile.Parse(&buf)
		if err != nil {
			return err
		}
		err = server.addProfile(ctx, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func (server Server) compatibleProfile(p *profile.Profile) bool {
	if len(server.profiles) == 0 {
		return true
	}

	if !equalValueType(server.profiles[0].PeriodType, p.PeriodType) {
		_ = server.log("Incompatible period types %v and %v", server.profiles[0].PeriodType, p.PeriodType)
		return false
	}

	if len(server.profiles[0].SampleType) != len(p.SampleType) {
		_ = server.log("Incompatible sample types %v and %v", server.profiles[0].SampleType, p.SampleType)
		return false
	}

	for i := range server.profiles[0].SampleType {
		if !equalValueType(server.profiles[0].SampleType[i], p.SampleType[i]) {
			_ = server.log("Incompatible sample types %v and %v", server.profiles[0].SampleType, p.SampleType)
			return false
		}
	}
	return true
}

func (server *Server) addProfile(context context.Context, p *profile.Profile) error {
	if !server.compatibleProfile(p) {
		err := server.log("Not compatible with previous profile(s). Clearing profile cache.")
		if err != nil {
			return err
		}

		_, err = server.ClearProfileCache(context, &empty.Empty{})
		if err != nil {
			return err
		}
	}

	server.profiles = append(server.profiles, p)
	return nil
}

// SetGRPCServerOption will set new GRPC Server option to GRPC Profile Server
func (server *Server) SetGRPCServerOption(grpcServerOption grpc.ServerOption) {
	server.grpcServerOptions = append(server.grpcServerOptions, grpcServerOption)
}

// NewServer method will create a new GRPC Profile Server instance
func NewServer(logger io.Writer, grpcServerOptions ...grpc.ServerOption) *Server {
	server := Server{Logger: logger}
	for _, serverOption := range grpcServerOptions {
		server.SetGRPCServerOption(serverOption)
	}
	return &server
}

// Serve the GRPC Profile server
func (server *Server) Serve(address string) error {
	listen, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	server.listen = listen

	err = server.log("Server listening at: ", server.listen.Addr)
	if err != nil {
		return err
	}

	server.server = grpc.NewServer(server.grpcServerOptions...)
	RegisterProfileServiceServer(server.server, server)
	reflection.Register(server.server)

	// Serve the server in go-routine
	go func() {
		_ = server.server.Serve(server.listen)
	}()

	return nil
}

// Stop GRPC Profile server
func (server *Server) Stop() error {
	server.server.Stop()
	return server.listen.Close()
}

// ClearProfileCache will clear all cached profiles
func (server *Server) ClearProfileCache(context.Context, *empty.Empty) (*Status, error) {
	server.profiles = nil
	return &Status{Code: StatusCode_OK}, nil
}

// SetMemProfileRate will set the rate of Memory Profiler
func (server *Server) SetMemProfileRate(_ context.Context, rate *Rate) (*Status, error) {
	if server.oldMemProfileRate == 0 {
		server.oldMemProfileRate = runtime.MemProfileRate
	}
	runtime.MemProfileRate = int(rate.Value)
	return &Status{Code: StatusCode_OK}, nil
}

// SetMutexProfileFraction will set the mutex profile fraction
func (server *Server) SetMutexProfileFraction(_ context.Context, rate *Rate) (*Status, error) {
	if server.oldMutexProfileFraction == 0 {
		server.oldMutexProfileFraction = runtime.SetMutexProfileFraction(int(rate.Value))
	} else {
		_ = runtime.SetMutexProfileFraction(int(rate.Value))
	}
	return &Status{Code: StatusCode_OK}, nil
}

// SetBlockProfileRate will set the block profile rate
func (server *Server) SetBlockProfileRate(_ context.Context, rate *Rate) (*Status, error) {
	runtime.SetBlockProfileRate(int(rate.Value))
	return &Status{Code: StatusCode_OK}, nil
}

// ResetMemProfileRate will set the memory profile rate to default value
func (server *Server) ResetMemProfileRate(context.Context, *empty.Empty) (*Status, error) {
	if server.oldMemProfileRate != 0 {
		runtime.MemProfileRate = server.oldMemProfileRate
	}
	return &Status{Code: StatusCode_OK}, nil
}

// ResetMutexProfileFraction will set the mutex profile fraction to default value
func (server *Server) ResetMutexProfileFraction(context.Context, *empty.Empty) (*Status, error) {
	runtime.SetMutexProfileFraction(server.oldMutexProfileFraction)
	return &Status{Code: StatusCode_OK}, nil
}

// ResetBlockProfileRate will set the block profile rate to default value
func (server *Server) ResetBlockProfileRate(context.Context, *empty.Empty) (*Status, error) {
	runtime.SetBlockProfileRate(0)
	return &Status{Code: StatusCode_OK}, nil
}

// CPU function will collect CPU profile
func (server *Server) CPU(ctx context.Context, duration *duration.Duration) (*Status, error) {
	err := server.log("Enabling CPU profiling")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}

	var buf bytes.Buffer
	startTime := time.Now()
	err = pprof.StartCPUProfile(&buf)
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}

	d, err := ptypes.Duration(duration)
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}

	endChan := time.After(d - time.Now().Sub(startTime))

	select {
	case <-endChan:
		break
	case <-ctx.Done():
		err = server.log("CPU profiling terminated early")
		if err != nil {
			return &Status{Code: StatusCode_Failed}, err
		}
		break
	}

	pprof.StopCPUProfile()
	err = server.log("Disabling CPU profiling")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}

	p, err := profile.Parse(&buf)
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	err = server.addProfile(ctx, p)
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	return &Status{Code: StatusCode_OK}, nil
}

// Memory function will collect Memory profile
func (server *Server) Memory(ctx context.Context, _ *empty.Empty) (*Status, error) {
	if runtime.MemProfileRate == 0 {
		return &Status{Code: StatusCode_Failed}, server.err("memory profiling is disabled")
	}
	err := server.collectLookupProfile(ctx, "heap")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	err = server.log("Memory profile collected")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	return &Status{Code: StatusCode_OK}, nil
}

// Mutex function will collect Mutex profile
func (server *Server) Mutex(ctx context.Context, _ *empty.Empty) (*Status, error) {
	err := server.collectLookupProfile(ctx, "mutex")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	err = server.log("Mutex profile collected")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	return &Status{Code: StatusCode_OK}, nil
}

// Block function will collect Block profile
func (server *Server) Block(ctx context.Context, _ *empty.Empty) (*Status, error) {
	err := server.collectLookupProfile(ctx, "block")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	err = server.log("Block profile collected")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	return &Status{Code: StatusCode_OK}, nil
}

// ThreadCreate function will collect ThreadCreate profile
func (server *Server) ThreadCreate(ctx context.Context, _ *empty.Empty) (*Status, error) {
	err := server.collectLookupProfile(ctx, "threadcreate")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	err = server.log("Thread create profile collected")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	return &Status{Code: StatusCode_OK}, nil
}

// GoRoutine function will collect GoRoutine profile
func (server *Server) GoRoutine(ctx context.Context, _ *empty.Empty) (*Status, error) {
	err := server.collectLookupProfile(ctx, "goroutine")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	err = server.log("Go routine profile collected")
	if err != nil {
		return &Status{Code: StatusCode_Failed}, err
	}
	return &Status{Code: StatusCode_OK}, nil
}

// Trace function will collect trace data and send it to client
func (server *Server) Trace(duration *duration.Duration, stream ProfileService_TraceServer) error {
	err := server.log("Enabling Trace")
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	startTime := time.Now()
	err = trace.Start(&buf)
	if err != nil {
		return err
	}

	d, err := ptypes.Duration(duration)
	if err != nil {
		return err
	}

	endChan := time.After(d - time.Now().Sub(startTime))

	select {
	case <-endChan:
		break
	case <-stream.Context().Done():
		err = server.log("Trace terminated early")
		if err != nil {
			return err
		}
		break
	}

	trace.Stop()
	err = server.log("Disabling Trace")
	if err != nil {
		return err
	}

	err = sendFileChunk(&buf, stream)
	if err != nil {
		return err
	}

	err = server.log("Sent trace to client")
	if err != nil {
		return err
	}

	return nil
}

// Download function will send the collected profile data to client
func (server *Server) Download(_ *empty.Empty, stream ProfileService_DownloadServer) error {
	// Stop profiling if not
	trace.Stop()
	pprof.StopCPUProfile()

	var buf bytes.Buffer

	p, err := profile.Merge(server.profiles)
	if err != nil {
		return err
	}

	err = p.Write(&buf)
	if err != nil {
		return err
	}

	_, err = server.ClearProfileCache(stream.Context(), &empty.Empty{})
	if err != nil {
		return err
	}

	err = sendFileChunk(&buf, stream)
	if err != nil {
		return err
	}

	err = server.log("Sent profile data to client")
	if err != nil {
		return err
	}

	return nil
}
