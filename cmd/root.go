package cmd

import (
	"fmt"
	config2 "github.com/Lavoaster/cloudsmith-sync/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var cfgFile string
var config *config2.Config
var workingDirectory string

func init() {
	wd, err := os.Getwd()
	workingDirectory = wd

	if err != nil {
		fmt.Println("Error getting working directory")
		os.Exit(1)
	}

	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", workingDirectory+"/config.yaml", "config file location")
}

func initConfig() {
	viper.SetConfigFile(cfgFile)

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}

	config = config2.NewConfigFromViper(workingDirectory)
	config.EnsureDirsExist()
}

var rootCmd = &cobra.Command{
	Use:   "cloudsmith-sync",
	Short: "Syncs composer git repositories and publishes them as versions on cloudsmith",
	Run:   func(cmd *cobra.Command, args []string) {},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func exitOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
