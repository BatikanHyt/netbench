package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/BatikanHyt/netbench/pkg/protocols"
	"github.com/spf13/cobra"
)

var rootCmdArgs struct {
	ConfigFile string
}
var runner = &protocols.Runner{}

var rootCmd = &cobra.Command{
	Use:               "netbench",
	Short:             "netbench is a network benchmark tool for http/s, ldap, smtp",
	Long:              `netbench is a network benchmark tool for http/s, ldap, smtp"`,
	Version:           "v0.1.0",
	PersistentPreRunE: validateRootArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if rootCmdArgs.ConfigFile == "" {
			fmt.Println("Please select config file")
			return
		}
		initConfig()

		runner.Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&rootCmdArgs.ConfigFile, "config", "", "Json format config file to load settings")
	rootCmd.PersistentFlags().IntVarP(&runner.Concurency, "concurency", "c", 1, "Number of concurent connection size")
	rootCmd.PersistentFlags().IntVarP(&runner.TotalRequest, "treq", "n", 1, "Number of total request to send")
	rootCmd.PersistentFlags().StringVarP(&runner.Duration, "duration", "d", "0s", "total duration 1s, 1m, 500ms etc")
}

func initConfig() {
	jsonFile, err := os.Open(rootCmdArgs.ConfigFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	jsonData, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal([]byte(jsonData), &runner)
}

func validateRootArgs(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("duration") && cmd.Flags().Changed("treq") {
		return fmt.Errorf("Cant set both duration(d) and total request(n)")
	}
	_, isValid := time.ParseDuration(runner.Duration)
	if isValid != nil {
		return fmt.Errorf("Error : %e", isValid)
	}
	if rootCmdArgs.ConfigFile != "" && cmd.Name() != "netbench" {
		return fmt.Errorf("Cannot use subcommand when config file flag used!")
	}

	return nil
}
