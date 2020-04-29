package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	profile "github.com/chanchal1987/grpc-profile"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	exe, _         = os.Executable()
	home, _        = os.UserHomeDir()
	applName       = filepath.Base(exe)
	applShortUsage = ""
	applLongUsage  = ""

	client          *profile.Client
	clientConnected bool

	defaultcfgFile = filepath.Join(home, "."+applName+".yaml")

	invalidArgumentsError = errors.New("invalid argument(s)")

	cfgFile string
	rootCmd = &cobra.Command{
		Use:   applName,
		Short: applShortUsage,
		Long:  applLongUsage,
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if clientConnected {
				err := client.Stop()
				if err != nil {
					return err
				}
			}
			return viper.WriteConfig()
		},
	}
)

// Execute rootCmd
func Execute(version, build string) error {
	if version != "" && build != "" {
		rootCmd.Version = version + ".b" + build
	}
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/."+applName+")")
	rootCmd.PersistentFlags().StringP("server", "s", "", "Address of the remote server where agent is running")
	rootCmd.PersistentFlags().String("cert", "", "Path to the TLS certificate. This will enable TLS authnetication")
	if err := viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server")); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("cert", rootCmd.PersistentFlags().Lookup("cert")); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Create config if not exists
		f, err := os.OpenFile(defaultcfgFile, os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		err = f.Close()
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		// Add config file to viper
		viper.SetConfigFile(defaultcfgFile)
	}
	viper.SetEnvPrefix(applName)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func connect(cmd *cobra.Command, args []string) error {
	address := viper.GetString("server")
	cert := viper.GetString("cert")
	if address == "" {
		return errors.New("please set server using global flag '--server'")
	}
	var options []*profile.DialOption

	if cert != "" {
		options = append(options, profile.DialAuthTypeTLS(cert))
	}
	var err error
	client, err = profile.NewClient(cmd.Context(), address, options...)
	if err != nil {
		return err
	}
	clientConnected = true
	return nil
}
