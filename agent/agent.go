package agent

//go:generate protoc -I ../proto/ ../proto/profile.proto --go_out=plugins=grpc:../proto

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/user"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"time"

	"github.com/chanchal1987/grpc-profile/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	timestamppb "github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

var lookupStr = map[proto.LookupProfile]string{
	proto.LookupProfile_profileTypeHeap:         "heap",
	proto.LookupProfile_profileTypeMutex:        "mutex",
	proto.LookupProfile_profileTypeBlock:        "block",
	proto.LookupProfile_profileTypeThreadCreate: "threadcreate",
	proto.LookupProfile_profileTypeGoRoutine:    "goroutine",
}

// Agent will store GRPC Profile Agent instance. We can create a instance of the agent using `NewAgent()` function
type Agent struct {
	listen        net.Listener
	server        *grpc.Server
	serverOptions []grpc.ServerOption
}

// NewAgent function will create a GRPC Profile Agent instance
func NewAgent(options ...*ServerOption) (agent *Agent, err error) {
	agent = &Agent{}
	err = agent.SetOptions(options...)
	if err != nil {
		return
	}
	return
}

// Start function will start GRPC Profile Agent
func (agent *Agent) Start(serverAddress string) (addr *net.TCPAddr, err error) {
	agent.listen, err = net.Listen("tcp", serverAddress)
	if err != nil {
		return
	}
	addr = agent.listen.Addr().(*net.TCPAddr)
	agent.server = grpc.NewServer(agent.serverOptions...)
	proto.RegisterProfileServiceServer(agent.server, agent)
	reflection.Register(agent.server)

	go func() {
		_ = agent.server.Serve(agent.listen)
	}()

	return
}

// Stop function will stop GRPC Profile Agent
func (agent *Agent) Stop() {
	agent.server.Stop()
}

// SetOption function will be used to set `ServerOption` to GRPC Profile Agent
func (agent *Agent) SetOption(option *ServerOption) error {
	if option == nil {
		return nil
	}
	if option.error != nil {
		return option.error
	}
	agent.serverOptions = append(agent.serverOptions, option.option)
	return nil
}

// SetOptions function will be used to set `ServerOption`s to GRPC Profile Agent
func (agent *Agent) SetOptions(options ...*ServerOption) (err error) {
	for _, option := range options {
		err = agent.SetOptions(option)
		if err != nil {
			return
		}
	}
	return
}

// ServerOption will create a Option for the GRPC Profile Agent
type ServerOption struct {
	option grpc.ServerOption
	error  error
}

// ServerAuthTypeInsecure function will create a Insecure Auth type GRPC Profile Agent option
func ServerAuthTypeInsecure() *ServerOption {
	return nil
}

// ServerAuthTypeTLS function will create a TLS Secure Auth type GRPC Profile Agent option
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
func (agent *Agent) Ping(context.Context, *empty.Empty) (*proto.StringType, error) {
	return &proto.StringType{Message: "pong"}, nil
}

func getUserName(id int) (string, error) {
	user, err := user.LookupId(strconv.Itoa(id))
	if err != nil {
		return "", err
	}
	return user.Name, nil
}

func getGroupName(id int) (string, error) {
	group, err := user.LookupGroupId(strconv.Itoa(id))
	if err != nil {
		return "", err
	}
	return group.Name, nil
}

// GetInfo function will get the current information about the server.
func (agent *Agent) GetInfo(context.Context, *empty.Empty) (*proto.InfoType, error) {
	var executableLStat, executableStat os.FileInfo
	var executableLStatModTime, executableStatModTime *timestamppb.Timestamp
	executable, err := os.Executable()
	if err != nil {
		executable = "unknown"
	} else {
		executableLStat, err = os.Lstat(executable)
		if err != nil {
			executableLStatModTime, _ = ptypes.TimestampProto(executableLStat.ModTime())
		}
		executableStat, _ = os.Stat(executable)
		if err != nil {
			executableStatModTime, _ = ptypes.TimestampProto(executableStat.ModTime())
		}
	}
	uid := os.Getuid()
	uidName, err := getUserName(uid)
	if err != nil {
		uidName = "unknown"
	}
	gid := os.Getgid()
	gidName, err := getGroupName(gid)
	if err != nil {
		gidName = "unknown"
	}
	euid := os.Geteuid()
	euidName, err := getUserName(euid)
	if err != nil {
		euidName = "unknown"
	}
	egid := os.Getegid()
	egidName, err := getGroupName(egid)
	if err != nil {
		egidName = "unknown"
	}
	wd, err := os.Getwd()
	if err != nil {
		wd = "unknown"
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		userCacheDir = "unknown"
	}
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = "unknown"
	}
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		userHomeDir = "unknown"
	}
	groupIDs, err := os.Getgroups()
	if err != nil {
		groupIDs = nil
	}
	var groups []*proto.IDName
	for _, group := range groupIDs {
		groupName, err := getGroupName(gid)
		if err != nil {
			groupName = "unknown"
		}
		groups = append(groups, &proto.IDName{
			ID:   int32(group),
			Name: groupName,
		})
	}
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	var lastGC, lastPause *timestamppb.Timestamp
	lastGC, err = ptypes.TimestampProto(time.Unix(0, int64(memStats.LastGC)))
	if err != nil {
		lastGC, _ = ptypes.TimestampProto(time.Unix(0, 0))
	}
	lastPause, err = ptypes.TimestampProto(time.Unix(0, int64(memStats.PauseNs[(memStats.NumGC+255)%256])))
	if err != nil {
		lastPause, _ = ptypes.TimestampProto(time.Unix(0, 0))
	}

	return &proto.InfoType{
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		GOMAXPROCS:   int32(runtime.GOMAXPROCS(0)),
		NumCPU:       int32(runtime.NumCPU()),
		NumCgoCall:   int32(runtime.NumCgoCall()),
		NumGoroutine: int32(runtime.NumGoroutine()),
		Version:      runtime.Version(),
		ProcessStats: &proto.ProcessStats{
			Environ:    os.Environ(),
			Executable: executable,
			ExecutableLStat: &proto.FileInfo{
				Name:     executableLStat.Name(),
				Size:     executableLStat.Size(),
				Mode:     uint32(executableLStat.Mode()),
				ModeTime: executableLStatModTime,
			},
			ExecutableStat: &proto.FileInfo{
				Name:     executableStat.Name(),
				Size:     executableStat.Size(),
				Mode:     uint32(executableStat.Mode()),
				ModeTime: executableStatModTime,
			},
			UID: &proto.IDName{
				ID:   int32(uid),
				Name: uidName,
			},
			GID: &proto.IDName{
				ID:   int32(gid),
				Name: gidName,
			},
			EUID: &proto.IDName{
				ID:   int32(euid),
				Name: euidName,
			},
			EGID: &proto.IDName{
				ID:   int32(egid),
				Name: egidName,
			},
			Groups:        groups,
			PageSize:      int32(os.Getpagesize()),
			PID:           int32(os.Getpid()),
			PPID:          int32(os.Getppid()),
			WD:            wd,
			Hostname:      hostname,
			UserCacheDir:  userCacheDir,
			UserConfigDir: userConfigDir,
			UserHomeDir:   userHomeDir,
		},
		MemStats: &proto.MemStats{
			Alloc:        memStats.Alloc,
			TotalAlloc:   memStats.TotalAlloc,
			Sys:          memStats.Sys,
			Lookups:      memStats.Lookups,
			Mallocs:      memStats.Mallocs,
			Frees:        memStats.Frees,
			HeapAlloc:    memStats.HeapAlloc,
			HeapSys:      memStats.HeapSys,
			HeapIdle:     memStats.HeapIdle,
			HeapInuse:    memStats.HeapInuse,
			HeapReleased: memStats.HeapReleased,
			HeapObjects:  memStats.HeapObjects,
			StackInuse:   memStats.StackInuse,
			StackSys:     memStats.StackSys,
			MSpanInuse:   memStats.MSpanInuse,
			MSpanSys:     memStats.MSpanSys,
			MCacheInuse:  memStats.MCacheInuse,
			MCacheSys:    memStats.MCacheSys,
			BuckHashSys:  memStats.BuckHashSys,
			GCSys:        memStats.GCSys,
			OtherSys:     memStats.OtherSys,
			NextGC:       memStats.NextGC,
			LastGC:       lastGC,
			PauseTotalNs: ptypes.DurationProto(time.Duration(memStats.PauseTotalNs)),
			LastPause:    lastPause,
			NumGC:        memStats.NumGC,
			NumForcedGC:  memStats.NumForcedGC,
		},
		MemProfileRate: int32(runtime.MemProfileRate),
	}, nil
}

// BinaryDump function get the dump of the current binary
func (agent *Agent) BinaryDump(_ *empty.Empty, profileServer proto.ProfileService_BinaryDumpServer) error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = bufio.NewReader(f).WriteTo(&grpcStreamWriter{profileServer})
	return err
}

// Set function will set the GRPC Profile Variable
func (agent *Agent) Set(_ context.Context, inputType *proto.SetProfileInputType) (*proto.IntType, error) {
	retValue := int32(-1)
	switch inputType.Variable {
	case proto.ProfileVariable_MemProfileRate:
		retValue = int32(runtime.MemProfileRate)
		runtime.MemProfileRate = int(inputType.Rate)
	case proto.ProfileVariable_CPUProfileRate:
		runtime.SetCPUProfileRate(int(inputType.Rate))
	case proto.ProfileVariable_MutexProfileFraction:
		retValue = int32(runtime.SetMutexProfileFraction(int(inputType.Rate)))
	case proto.ProfileVariable_BlockProfileRate:
		runtime.SetBlockProfileRate(int(inputType.Rate))
	}
	return &proto.IntType{Value: retValue}, nil
}

// GC function will run GC on remote agent
func (agent *Agent) GC(context.Context, *empty.Empty) (*empty.Empty, error) {
	runtime.GC()
	return &empty.Empty{}, nil
}

// LookupProfile will run a profile for lookup pprof type
func (agent *Agent) LookupProfile(inputType *proto.LookupProfileInputType, profileServer proto.ProfileService_LookupProfileServer) error {
	prof := pprof.Lookup(lookupStr[inputType.ProfileType])
	if prof == nil {
		return nil
	}

	err := prof.WriteTo(&grpcStreamWriter{profileServer}, 0)
	if err != nil {
		return err
	}
	return nil
}

func (agent *Agent) runNonLookup(ctx context.Context, startFunc func(io.Writer) error, stopFunc func(), duration time.Duration, writer io.Writer) error {
	startTime := time.Now()
	err := startFunc(writer)
	if err != nil {
		return err
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, duration-time.Since(startTime))
	defer cancel()
	<-timeoutCtx.Done()
	stopFunc()
	return nil
}

// NonLookupProfile will run a profile for non lookup pprof type
func (agent *Agent) NonLookupProfile(inputType *proto.NonLookupProfileInputType, profileServer proto.ProfileService_NonLookupProfileServer) error {
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
		return errors.New("unknown profile type")
	}

	dur, err := ptypes.Duration(inputType.Duration)
	if err != nil {
		return err
	}

	writer := grpcStreamWriter{profileServer}
	err = agent.runNonLookup(profileServer.Context(), startFunc, stopFunc, dur, &writer)
	if err != nil {
		return err
	}
	return nil
}

// StopNonLookupProfile will stop non lookup profile type (if running)
func (agent *Agent) StopNonLookupProfile(_ context.Context, profileType *proto.NonLookupProfileType) (*empty.Empty, error) {
	switch profileType.Profile {
	case proto.NonLookupProfile_profileTypeCPU:
		pprof.StopCPUProfile()
	case proto.NonLookupProfile_profileTypeTrace:
		trace.Stop()
	default:
		return &empty.Empty{}, errors.New("unknown profile type")
	}
	return &empty.Empty{}, nil
}
