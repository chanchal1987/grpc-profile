package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/chanchal1987/grpc-profile/agent"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

func init() {
	rootCmd.AddCommand(dummyCmd)
}

var dummyCmd = &cobra.Command{
	Use:       "dummy-agent [server-address [duration]]",
	Short:     "Start a dummy agent",
	Long:      `A dummy agent will start. if you want oto run it in background please start with nohup pf start using '&'`,
	Example:   applName + " dummy-agent\n" + applName + " dummy-agent \"0.0.0.0:8080\"\n" + applName + " dummy-agent \":8080\" 5s",
	ValidArgs: nil,
	RunE: func(cmd *cobra.Command, args []string) error {
		var addr string
		var dur string

		if len(args) >= 1 {
			addr = args[0]
		}

		if len(args) >= 2 {
			dur = args[1]
		}

		server, err := agent.NewAgent()
		if err != nil {
			return err
		}

		tcpAddr, err := server.Start(addr)
		if err != nil {
			return err
		}

		fmt.Println("Dummy agent started at:", tcpAddr)

		defer func() {
			fmt.Println("Dummy agent is stopping...")
			server.Stop()
		}()

		ctx, calcelFunc := context.WithCancel(cmd.Context())

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)

		go func() {
			select {
			case <-sigChan:
				calcelFunc()
			case <-ctx.Done():
				break
			}
		}()

		if dur != "" {
			var pDur time.Duration
			pDur, err = time.ParseDuration(dur)
			if err == nil {
				fmt.Println("Dummy agent will stop automatically after:", pDur)
				ctx, calcelFunc = context.WithTimeout(ctx, pDur)
			}
		}

		// Add some load to agent
		for i := 0; i < runtime.NumCPU(); i++ {
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					default:
						continue
					}
				}
			}()
		}
		<-ctx.Done()
		calcelFunc()
		return nil
	},
}
