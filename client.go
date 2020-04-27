package profile

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/chanchal1987/grpc-profile/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
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
				err = nil
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

	// CPUProfRate controls CPU profiling rate to hz samples per second. If hz <= 0, SetCPUProfileRate turns off
	// profiling. If the profiler is on, the rate cannot be changed without first turning it off.
	CPUProfRate

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

// FileInfo will store informarion about a file
type FileInfo struct {
	Name     string
	Size     int64
	Mode     uint
	ModeTime time.Time
}

// IDName will store User / Group ID and Name
type IDName struct {
	ID   int
	Name string
}

// ProcessStats will store status about the process
type ProcessStats struct {
	Environ         []string
	Executable      string
	ExecutableLStat FileInfo
	ExecutableStat  FileInfo
	UID             IDName
	GID             IDName
	EUID            IDName
	EGID            IDName
	Groups          []IDName
	PageSize        int
	PID             int
	PPID            int
	WD              string
	Hostname        string
	UserCacheDir    string
	UserConfigDir   string
	UserHomeDir     string
}

// MemStats will store status about the memory
type MemStats struct {
	Alloc        uint64
	TotalAlloc   uint64
	Sys          uint64
	Lookups      uint64
	Mallocs      uint64
	Frees        uint64
	HeapAlloc    uint64
	HeapSys      uint64
	HeapIdle     uint64
	HeapInuse    uint64
	HeapReleased uint64
	HeapObjects  uint64
	StackInuse   uint64
	StackSys     uint64
	MSpanInuse   uint64
	MSpanSys     uint64
	MCacheInuse  uint64
	MCacheSys    uint64
	BuckHashSys  uint64
	GCSys        uint64
	OtherSys     uint64
	NextGC       uint64
	LastGC       time.Time
	PauseTotalNs time.Duration
	LastPause    time.Time
	NumGC        uint32
	NumForcedGC  uint32
}

// InfoType will store all informations about the agent
type InfoType struct {
	GOOS           string
	GOARCH         string
	GOMAXPROCS     int
	NumCPU         int
	NumCgoCall     int
	NumGoroutine   int
	Version        string
	ProcessStats   ProcessStats
	MemStats       MemStats
	MemProfileRate int
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
	client = &Client{}
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

// GetInfo function will get current information about the agent
func (client *Client) GetInfo(ctx context.Context) (*InfoType, error) {
	info, err := client.client.GetInfo(ctx, &empty.Empty{}, client.callOptions...)
	if err != nil {
		return nil, err
	}
	var modTimeL, modTime, lastGC, lastPause time.Time
	var pauseTotalNs time.Duration
	if info.ProcessStats.ExecutableLStat.ModeTime == nil {
		modTimeL = time.Unix(0, 0)
	} else {
		modTimeL, err = ptypes.Timestamp(info.ProcessStats.ExecutableLStat.ModeTime)
		if err != nil {
			return nil, err
		}
	}

	if info.ProcessStats.ExecutableStat.ModeTime == nil {
		modTime = time.Unix(0, 0)
	} else {
		modTime, err = ptypes.Timestamp(info.ProcessStats.ExecutableStat.ModeTime)
		if err != nil {
			return nil, err
		}
	}
	var groups []IDName
	for _, g := range info.ProcessStats.Groups {
		groups = append(groups, IDName{ID: int(g.ID), Name: g.Name})
	}
	if info.MemStats.LastGC == nil {
		lastGC = time.Unix(0, 0)
	} else {
		lastGC, err = ptypes.Timestamp(info.MemStats.LastGC)
		if err != nil {
			return nil, err
		}
	}
	if info.MemStats.PauseTotalNs != nil {
		pauseTotalNs, err = ptypes.Duration(info.MemStats.PauseTotalNs)
		if err != nil {
			return nil, err
		}
	}
	if info.MemStats.LastPause == nil {
		lastPause = time.Unix(0, 0)
	} else {
		lastPause, err = ptypes.Timestamp(info.MemStats.LastPause)
		if err != nil {
			return nil, err
		}
	}

	return &InfoType{
		GOOS:         info.GOOS,
		GOARCH:       info.GOARCH,
		GOMAXPROCS:   int(info.GOMAXPROCS),
		NumCPU:       int(info.NumCPU),
		NumCgoCall:   int(info.NumCgoCall),
		NumGoroutine: int(info.NumGoroutine),
		Version:      info.Version,
		ProcessStats: ProcessStats{
			Environ:    info.ProcessStats.Environ,
			Executable: info.ProcessStats.Executable,
			ExecutableLStat: FileInfo{
				Name:     info.ProcessStats.ExecutableLStat.Name,
				Size:     info.ProcessStats.ExecutableLStat.Size,
				Mode:     uint(info.ProcessStats.ExecutableLStat.Mode),
				ModeTime: modTimeL,
			},
			ExecutableStat: FileInfo{
				Name:     info.ProcessStats.ExecutableStat.Name,
				Size:     info.ProcessStats.ExecutableStat.Size,
				Mode:     uint(info.ProcessStats.ExecutableStat.Mode),
				ModeTime: modTime,
			},
			UID: IDName{
				ID:   int(info.ProcessStats.UID.ID),
				Name: info.ProcessStats.UID.Name,
			},
			GID: IDName{
				ID:   int(info.ProcessStats.GID.ID),
				Name: info.ProcessStats.GID.Name,
			},
			EUID: IDName{
				ID:   int(info.ProcessStats.EUID.ID),
				Name: info.ProcessStats.EUID.Name,
			},
			EGID: IDName{
				ID:   int(info.ProcessStats.EGID.ID),
				Name: info.ProcessStats.EGID.Name,
			},
			Groups:        groups,
			PageSize:      int(info.ProcessStats.PageSize),
			PID:           int(info.ProcessStats.PID),
			PPID:          int(info.ProcessStats.PPID),
			WD:            info.ProcessStats.WD,
			Hostname:      info.ProcessStats.Hostname,
			UserCacheDir:  info.ProcessStats.UserCacheDir,
			UserConfigDir: info.ProcessStats.UserConfigDir,
			UserHomeDir:   info.ProcessStats.UserHomeDir,
		},
		MemStats: MemStats{
			Alloc:        info.MemStats.Alloc,
			TotalAlloc:   info.MemStats.TotalAlloc,
			Sys:          info.MemStats.Sys,
			Lookups:      info.MemStats.Lookups,
			Mallocs:      info.MemStats.Mallocs,
			Frees:        info.MemStats.Frees,
			HeapAlloc:    info.MemStats.HeapAlloc,
			HeapSys:      info.MemStats.HeapSys,
			HeapIdle:     info.MemStats.HeapIdle,
			HeapInuse:    info.MemStats.HeapInuse,
			HeapReleased: info.MemStats.HeapReleased,
			HeapObjects:  info.MemStats.HeapObjects,
			StackInuse:   info.MemStats.StackInuse,
			StackSys:     info.MemStats.StackSys,
			MSpanInuse:   info.MemStats.MSpanInuse,
			MSpanSys:     info.MemStats.MSpanSys,
			MCacheInuse:  info.MemStats.MCacheInuse,
			MCacheSys:    info.MemStats.MCacheSys,
			BuckHashSys:  info.MemStats.BuckHashSys,
			GCSys:        info.MemStats.GCSys,
			OtherSys:     info.MemStats.OtherSys,
			NextGC:       info.MemStats.NextGC,
			LastGC:       lastGC,
			PauseTotalNs: pauseTotalNs,
			LastPause:    lastPause,
			NumGC:        info.MemStats.NumGC,
			NumForcedGC:  info.MemStats.NumForcedGC,
		},
		MemProfileRate: int(info.MemProfileRate),
	}, nil
}

// BinaryDump function will get a binary dump of the remote binary
func (client *Client) BinaryDump(ctx context.Context, writer io.Writer) error {
	stream, err := client.client.BinaryDump(ctx, &empty.Empty{}, client.callOptions...)
	if err != nil {
		return err
	}
	return receiveFileChunk(writer, stream)
}

// Set function will set the GRPC Profile Variable
func (client *Client) Set(ctx context.Context, v Variable, r int) (int, error) {
	val, err := client.client.Set(ctx, &proto.SetProfileInputType{Variable: lookupVariable[v], Rate: int32(r)}, client.callOptions...)
	if err != nil {
		return 0, err
	}
	return int(val.Value), nil
}

// GC function will run GC on remote server
func (client *Client) GC(ctx context.Context) error {
	_, err := client.client.GC(ctx, &empty.Empty{}, client.callOptions...)
	if err != nil {
		return err
	}
	return nil
}

// LookupProfile will run a profile for lookup pprof type
func (client *Client) LookupProfile(ctx context.Context, t LookupType, writer io.Writer, keep bool) error {
	stream, err := client.client.LookupProfile(ctx, &proto.LookupProfileInputType{ProfileType: lookupLookupType[t]}, client.callOptions...)
	if err != nil {
		return err
	}
	return receiveFileChunk(writer, stream)
}

// NonLookupProfile will run a profile for non lookup pprof type
func (client *Client) NonLookupProfile(ctx context.Context, t NonLookupType, d time.Duration, writer io.Writer, wait, keep bool) error {
	stream, err := client.client.NonLookupProfile(ctx, &proto.NonLookupProfileInputType{ProfileType: lookupNonLookupType[t], Duration: ptypes.DurationProto(d)}, client.callOptions...)
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
