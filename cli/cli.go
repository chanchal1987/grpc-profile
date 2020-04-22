package main

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/abiosoft/ishell"
	profile "github.com/chanchal1987/grpc-profile"
)

type connectionStatus struct {
	Connected     bool
	serverAddress string
	client        *profile.Client
}

func (status *connectionStatus) Connect(ctx context.Context, serverAddress string) error {
	client, err := profile.NewClient(ctx, serverAddress)
	if err != nil {
		return err
	}
	status.serverAddress = serverAddress
	status.client = client
	status.Connected = true
	return nil
}

func (status *connectionStatus) Close() error {
	status.Connected = false
	return status.client.Stop()
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

var profVars = []string{"MemProfileRate", "MutexProfileFraction", "BlockProfileRate"}
var profTypes = []string{"Heap", "Mutex", "Block", "ThreadCreate", "GoRoutine", "CPU", "Trace"}
var nonLookupTypes = []string{"CPU", "Trace"}

var variableMap = map[string]profile.Variable{
	"MemProfileRate":       profile.MemProfRate,
	"MutexProfileFraction": profile.MutexProfileFraction,
	"BlockProfileRate":     profile.BlockProfileRate,
}
var lookupClientMap = map[string]profile.LookupType{
	"Heap":         profile.HeapType,
	"Mutex":        profile.MutexType,
	"Block":        profile.BlockType,
	"ThreadCreate": profile.ThreadCreateType,
	"GoRoutine":    profile.GoRoutineType,
}
var nonLookupClientMap = map[string]profile.NonLookupType{
	"CPU":   profile.CPUType,
	"Trace": profile.TraceType,
}

func commandConnect(ctx context.Context, conn *connectionStatus, err error) *ishell.Cmd {
	return &ishell.Cmd{
		Name:     "connect",
		Aliases:  []string{"c"},
		Help:     "Connect to a remote server",
		LongHelp: "Connect to a server and port where GRPC profile server is running",
		Func: func(c *ishell.Context) {
			var address string
			if len(c.Args) == 1 {
				address = c.Args[0]
			} else {
				c.Print("Server Address: ")
				address = c.ReadLine()
			}

			err = conn.Connect(ctx, address)
			if err != nil {
				c.Err(err)
			}
		},
	}
}

func commandSet(ctx context.Context, conn *connectionStatus, err error) *ishell.Cmd {
	return &ishell.Cmd{
		Name:     "set",
		Aliases:  []string{"s"},
		Help:     "Set GRPC Profile variable in remote server",
		LongHelp: "Set GRPC Profile variable in remote server",
		Func: func(c *ishell.Context) {
			if !conn.Connected {
				c.Err(errors.New("not connected. Please use \"connect\" command to connect"))
				return
			}

			var variable profile.Variable
			var rate int

			if len(c.Args) == 2 {
				var ok bool
				variable, ok = variableMap[c.Args[0]]
				if !ok {
					c.Err(errors.New("unknown variable"))
					return
				}
				rate, err = strconv.Atoi(c.Args[1])
				if err != nil {
					c.Err(err)
					return
				}
			} else {
				variable = profile.Variable(c.MultiChoice(profVars, "Which Variable?"))
				c.Print("Rate? ")
				rate, err = strconv.Atoi(c.ReadLine())
				if err != nil {
					c.Err(err)
					return
				}
			}

			err = conn.client.Set(ctx, variable, rate)
			if err != nil {
				c.Err(err)
			}
		},
		Completer: func(args []string) []string {
			if args == nil {
				return profVars
			}
			return nil
		},
	}
}

func commandReset(ctx context.Context, conn *connectionStatus, err error) *ishell.Cmd {
	return &ishell.Cmd{
		Name:     "reset",
		Aliases:  []string{"r"},
		Help:     "Reset GRPC Profile variable in remote server",
		LongHelp: "Reset GRPC Profile variable in remote server",
		Func: func(c *ishell.Context) {
			if !conn.Connected {
				c.Err(errors.New("not connected. Please use \"connect\" command to connect"))
				return
			}

			var variable profile.Variable

			if len(c.Args) == 1 {
				var ok bool
				variable, ok = variableMap[c.Args[0]]
				if !ok {
					c.Err(errors.New("unknown variable"))
					return
				}
			} else {
				variable = profile.Variable(c.MultiChoice(profVars, "Which Variable?"))
				c.Print("Rate? ")
			}

			err = conn.client.Reset(ctx, variable)
			if err != nil {
				c.Err(err)
			}
		},
		Completer: func(args []string) []string {
			if args == nil {
				return profVars
			}
			return nil
		},
	}
}

func commandProfile(ctx context.Context, conn *connectionStatus, err error) *ishell.Cmd {
	return &ishell.Cmd{
		Name:     "profile",
		Aliases:  []string{"p"},
		Help:     "Collect pprof from remote server",
		LongHelp: "Collect pprof from remote server",
		Func: func(c *ishell.Context) {
			if !conn.Connected {
				c.Err(errors.New("not connected. Please use \"connect\" command to connect"))
				return
			}

			var profileType string
			var dur time.Duration
			var nonLookup bool

			if len(c.Args) == 2 && contains(nonLookupTypes, c.Args[0]) {
				profileType = c.Args[0]
				dur, err = time.ParseDuration(c.Args[1])
				if err != nil {
					c.Err(err)
				}
				nonLookup = true
			} else if len(c.Args) == 1 && contains(profTypes, c.Args[0]) && !contains(nonLookupTypes, c.Args[0]) {
				profileType = c.Args[0]
				nonLookup = false
			} else {
				profileType = profTypes[c.MultiChoice(profTypes, "Which Profile?")]
				nonLookup = false
				if contains(nonLookupTypes, profileType) {
					c.Print("Duration? ")
					dur, err = time.ParseDuration(c.ReadLine())
					if err != nil {
						c.Err(err)
						return
					}
					nonLookup = true
				}
			}

			var file *os.File
			c.Print("File name to download the profile data: ")
			file, err = os.Create(c.ReadLine())
			if err != nil {
				c.Err(err)
				return
			}

			if nonLookup {
				err = conn.client.NonLookupProfile(ctx, nonLookupClientMap[profileType], dur, file, true, false)
			} else {
				err = conn.client.LookupProfile(ctx, lookupClientMap[profileType], file, false)
			}
			if err != nil {
				c.Err(err)
			}
		},
		Completer: func(args []string) []string {
			if args == nil {
				return profTypes
			}
			return nil
		},
	}
}

func commandDummyServer(ctx context.Context, conn *connectionStatus, err error) *ishell.Cmd {
	return &ishell.Cmd{
		Name:     "dummy-server",
		Help:     "Run dummy server for testing",
		LongHelp: "Run dummy server for testing",
		Func: func(c *ishell.Context) {
			server, err := profile.NewServer()
			if err != nil {
				c.Err(err)
				return
			}

			go func(server *profile.Server) {
				done := make(chan bool)
				defer func() {
					c.Println("Dummy server is stopping...")
					err = server.Stop()
					if err != nil {
						c.Err(err)
					}
				}()
				for i := 0; i < runtime.NumCPU(); i++ {
					go func() {
						for {
							select {
							case <-done:
								return
							default:
								continue
							}
						}
					}()
				}
				<-ctx.Done()
				close(done)
			}(server)

			addr, err := server.Start("")
			if err != nil {
				c.Err(err)
				return
			}
			c.Printf("Dummy server listening at: %v\n", addr)
		},
	}
}

func main() {
	var conn connectionStatus
	var err error
	ctx := context.Background()

	shell := ishell.New()
	shell.SetPrompt("gprof >> ")
	shell.Println("GRPC Profile Interactive Shell")

	shell.AddCmd(commandConnect(ctx, &conn, err))
	shell.AddCmd(commandSet(ctx, &conn, err))
	shell.AddCmd(commandReset(ctx, &conn, err))
	shell.AddCmd(commandProfile(ctx, &conn, err))
	shell.AddCmd(commandDummyServer(ctx, &conn, err))

	shell.Run()

	if conn.Connected {
		conn.Close()
	}
}
