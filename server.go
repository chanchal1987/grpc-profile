package profile

//go:generate protoc -I proto/ proto/profile.proto --go_out=plugins=grpc:proto

import (
	"bytes"
	"context"
	"io"
	"net"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"

	"github.com/chanchal1987/grpc-profile/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/pprof/profile"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"gopkg.in/errgo.v2/fmt/errors"
)

var lookupStr = map[proto.LookupProfile]string{
	proto.LookupProfile_profileTypeHeap:         "heap",
	proto.LookupProfile_profileTypeMutex:        "mutex",
	proto.LookupProfile_profileTypeBlock:        "block",
	proto.LookupProfile_profileTypeThreadCreate: "threadcreate",
	proto.LookupProfile_profileTypeGoRoutine:    "goroutine",
}

// Server will store GRPC Profile Server instance. We can create a instance of the server using `NewServer()` function
type Server struct {
	lookupProfile    map[proto.LookupProfile]*profile.Profile
	nonLookupProfile map[proto.NonLookupProfile]*profile.Profile
	initVariable     map[proto.ProfileVariable]int
	initializedVars  bool
	variable         map[proto.ProfileVariable]int
	profileRunning   bool
	listen           net.Listener
	server           *grpc.Server
	serverOptions    []grpc.ServerOption
}

// NewServer function will create a GRPC Profile Server instance
func NewServer(options ...*ServerOption) (server *Server, err error) {
	server = &Server{}
	err = server.SetOptions(options...)
	if err != nil {
		return
	}
	err = server.initVariables()
	return
}

// Start function will start GRPC Profile Server
func (server *Server) Start(serverAddress string) (addr *net.TCPAddr, err error) {
	server.listen, err = net.Listen("tcp", serverAddress)
	if err != nil {
		return
	}
	addr = server.listen.Addr().(*net.TCPAddr)
	server.server = grpc.NewServer(server.serverOptions...)
	proto.RegisterProfileServiceServer(server.server, server)
	reflection.Register(server.server)

	go func() {
		_ = server.server.Serve(server.listen)
	}()

	return
}

// Stop function will stop GRPC Profile Server
func (server *Server) Stop() error {
	server.server.Stop()
	return server.listen.Close()
}

// SetOption function will be used to set `ServerOption` to GRPC Profile Server
func (server *Server) SetOption(option *ServerOption) error {
	if option == nil {
		return nil
	}
	if option.error != nil {
		return option.error
	}
	server.serverOptions = append(server.serverOptions, option.option)
	return nil
}

// SetOptions function will be used to set `ServerOption`s to GRPC Profile Server
func (server *Server) SetOptions(options ...*ServerOption) (err error) {
	for _, option := range options {
		err = server.SetOptions(option)
		if err != nil {
			return
		}
	}
	return
}

func (server *Server) initVariables() error {
	if server.initializedVars {
		return errors.New("variables are already initialized")
	}

	if server.initVariable == nil {
		server.variable = make(map[proto.ProfileVariable]int)
	}

	server.initVariable[proto.ProfileVariable_MemProfileRate] = runtime.MemProfileRate

	muFrac := runtime.SetMutexProfileFraction(0)
	_ = runtime.SetMutexProfileFraction(muFrac)
	server.initVariable[proto.ProfileVariable_MutexProfileFraction] = muFrac
	server.initVariable[proto.ProfileVariable_BlockProfileRate] = 0

	server.variable = server.initVariable
	server.initializedVars = true
	return nil
}

// ServerOption will create a Option for the GRPC Profile Server
type ServerOption struct {
	option grpc.ServerOption
	error  error
}

// ServerAuthTypeInsecure function will create a Insecure Auth type GRPC Profile Server option
func ServerAuthTypeInsecure() *ServerOption {
	return nil
}

// ServerAuthTypeTLS function will create a TLS Secure Auth type GRPC Profile Server option
func ServerAuthTypeTLS(certFile, keyFile string) *ServerOption {
	cred, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		return &ServerOption{error: err}
	}
	return &ServerOption{option: grpc.Creds(cred)}
}

type grpcStreamWriter struct {
	Stream interface{ Send(*proto.FileChunk) error }
}

func (w *grpcStreamWriter) Write(bytes []byte) (n int, err error) {
	for _, b := range bytes {
		err = w.Stream.Send(&proto.FileChunk{Content: []byte{b}})
		if err != nil {
			return
		}
		n++
	}
	return
}

// Ping function will be used to test the connectivity to the server from client.
// This function will always return a response contains the word "pong"
func (server *Server) Ping(context.Context, *empty.Empty) (*proto.StringType, error) {
	return &proto.StringType{Message: "pong"}, nil
}

// ClearProfileCache function will clear all saved profiles in the GRPC Profile Server
func (server *Server) ClearProfileCache(_ context.Context, _ *empty.Empty) (*empty.Empty, error) {
	server.lookupProfile = nil
	server.nonLookupProfile = nil
	return &empty.Empty{}, nil
}

// Set function will set the GRPC Profile Variable
func (server *Server) Set(_ context.Context, inputType *proto.SetProfileInputType) (*empty.Empty, error) {
	if !server.initializedVars {
		return &empty.Empty{}, status.Error(codes.FailedPrecondition, "variables are not initialized yet")
	}

	server.variable[inputType.Variable] = int(inputType.Rate)
	switch inputType.Variable {
	case proto.ProfileVariable_MemProfileRate:
		runtime.MemProfileRate = server.variable[inputType.Variable]
	case proto.ProfileVariable_MutexProfileFraction:
		_ = runtime.SetMutexProfileFraction(server.variable[inputType.Variable])
	case proto.ProfileVariable_BlockProfileRate:
		runtime.SetBlockProfileRate(server.variable[inputType.Variable])
	}
	return &empty.Empty{}, nil
}

// Reset function will reset the GRPC Profile Variable to its original value
func (server *Server) Reset(_ context.Context, inputType *proto.ResetProfileInputType) (*empty.Empty, error) {
	if !server.initializedVars {
		return &empty.Empty{}, status.Error(codes.FailedPrecondition, "variables are not initialized yet")
	}

	rate := server.initVariable[inputType.Variable]
	server.variable[inputType.Variable] = rate
	switch inputType.Variable {
	case proto.ProfileVariable_MemProfileRate:
		runtime.MemProfileRate = rate
	case proto.ProfileVariable_MutexProfileFraction:
		_ = runtime.SetMutexProfileFraction(rate)
	case proto.ProfileVariable_BlockProfileRate:
		runtime.SetBlockProfileRate(rate)
	}
	return &empty.Empty{}, nil
}

// LookupProfile will run a profile for lookup pprof type
func (server *Server) LookupProfile(inputType *proto.LookupProfileInputType, profileServer proto.ProfileService_LookupProfileServer) (err error) {
	prof := pprof.Lookup(lookupStr[inputType.ProfileType])
	if prof == nil {
		return
	}

	writer := grpcStreamWriter{profileServer}
	if inputType.Keep {
		var buf bytes.Buffer
		err = prof.WriteTo(&buf, 0)
		if err != nil {
			return
		}
		_, err = writer.Write(buf.Bytes())
		if err != nil {
			return
		}
		var p *profile.Profile
		p, err = profile.Parse(&buf)
		if err != nil {
			return
		}
		if server.lookupProfile == nil {
			server.lookupProfile = make(map[proto.LookupProfile]*profile.Profile)
		}
		if _, ok := server.lookupProfile[inputType.ProfileType]; ok {
			p, err = profile.Merge([]*profile.Profile{server.lookupProfile[inputType.ProfileType], p})
			if err != nil {
				return
			}
		}
		server.lookupProfile[inputType.ProfileType] = p
	} else {
		err = prof.WriteTo(&writer, 0)
		if err != nil {
			return
		}
	}
	return
}

// DownloadLookupProfile will download a lookup profile type storred in GRPC Profile Server
func (server *Server) DownloadLookupProfile(profileType *proto.LookupProfileType, profileServer proto.ProfileService_DownloadLookupProfileServer) error {
	var ok bool
	var prof *profile.Profile
	if server.lookupProfile[profileType.Profile] == nil {
		ok = false
	}
	if ok {
		prof, ok = server.lookupProfile[profileType.Profile]
	}
	if !ok {
		return status.Error(codes.NotFound, "no profile data saved")
	}

	writer := grpcStreamWriter{profileServer}
	return prof.Write(&writer)
}

func (server *Server) runNonLookup(ctx context.Context, startFunc func(io.Writer) error, stopFunc func(), duration time.Duration, waitForCompletion bool, writer io.Writer) error {
	server.profileRunning = true
	startTime := time.Now()
	err := startFunc(writer)
	if err != nil {
		return err
	}
	timeoutCtx, cancelFunf := context.WithTimeout(ctx, duration-time.Since(startTime))
	var wg sync.WaitGroup
	wg.Add(1)

	go func(server *Server, ctx context.Context, stopFunc func(), cancelFunc context.CancelFunc) {
		defer wg.Done()
		<-ctx.Done()
		stopFunc()
		cancelFunc()
		server.profileRunning = false
	}(server, timeoutCtx, stopFunc, cancelFunf)
	if waitForCompletion {
		wg.Wait()
	}
	return nil
}

// NonLookupProfile will run a profile for non lookup pprof type
func (server *Server) NonLookupProfile(inputType *proto.NonLookupProfileInputType, profileServer proto.ProfileService_NonLookupProfileServer) error {
	var startFunc func(io.Writer) error
	var stopFunc func()

	switch inputType.ProfileType {
	case proto.NonLookupProfile_profileTypeCPU:
		startFunc = pprof.StartCPUProfile
		stopFunc = pprof.StopCPUProfile
	case proto.NonLookupProfile_profileTypeTrace:
		startFunc = trace.Start
		stopFunc = trace.Stop
	default:
		return status.Error(codes.NotFound, "unknown profile type")
	}

	dur, err := ptypes.Duration(inputType.Duration)
	if err != nil {
		return err
	}

	writer := grpcStreamWriter{profileServer}
	if inputType.Keep {
		var buf bytes.Buffer
		err := server.runNonLookup(profileServer.Context(), startFunc, stopFunc, dur, inputType.WaitForCompletion, &buf)
		if err != nil {
			return err
		}

		_, err = writer.Write(buf.Bytes())
		if err != nil {
			return err
		}

		p, err := profile.Parse(&buf)
		if err != nil {
			return err
		}

		if server.nonLookupProfile == nil {
			server.nonLookupProfile = make(map[proto.NonLookupProfile]*profile.Profile)
		}
		if _, ok := server.nonLookupProfile[inputType.ProfileType]; ok {
			p, err = profile.Merge([]*profile.Profile{server.nonLookupProfile[inputType.ProfileType], p})
			if err != nil {
				return err
			}
		}
		server.nonLookupProfile[inputType.ProfileType] = p
	} else {
		err := server.runNonLookup(profileServer.Context(), startFunc, stopFunc, dur, inputType.WaitForCompletion, &writer)
		if err != nil {
			return err
		}
	}
	return nil
}

// StopNonLookupProfile will stop non lookup profile type (if running)
func (server *Server) StopNonLookupProfile(_ context.Context, profileType *proto.NonLookupProfileType) (*empty.Empty, error) {
	switch profileType.Profile {
	case proto.NonLookupProfile_profileTypeCPU:
		pprof.StopCPUProfile()
	case proto.NonLookupProfile_profileTypeTrace:
		trace.Stop()
	default:
		return &empty.Empty{}, status.Error(codes.NotFound, "unknown profile type")
	}
	return &empty.Empty{}, nil
}

// DownloadNonLookupProfile will download a non lookup profile type storred in GRPC Profile Server
func (server *Server) DownloadNonLookupProfile(profileType *proto.NonLookupProfileType, profileServer proto.ProfileService_DownloadNonLookupProfileServer) error {
	var ok bool
	var prof *profile.Profile
	if server.nonLookupProfile[profileType.Profile] == nil {
		ok = false
	}
	if ok {
		prof, ok = server.nonLookupProfile[profileType.Profile]
	}
	if !ok {
		return status.Error(codes.NotFound, "no profile data saved")
	}

	writer := grpcStreamWriter{profileServer}
	return prof.Write(&writer)
}
