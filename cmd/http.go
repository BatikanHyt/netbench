package cmd

import (
	"errors"
	"fmt"

	"github.com/BatikanHyt/netbench/pkg/collector"
	"github.com/BatikanHyt/netbench/pkg/helpers"
	httpClient "github.com/BatikanHyt/netbench/pkg/protocols"

	"github.com/spf13/cobra"
)

var httpCmd = &cobra.Command{
	Use:     "http [URI]",
	Short:   "Print the version number of netbench",
	Long:    "Print the version number of netbench",
	PreRunE: valideHttpArgs,
	Run:     runHttpCmd,
}

var validMethod = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"}
var client = httpClient.NewHttpClient()

func init() {
	httpCmd.Flags().StringVarP(&client.Method, "method", "m", "GET", "Http method to use")
	httpCmd.Flags().StringToStringVarP(&client.Headers, "headers", "H", map[string]string{}, "Headers in key=value format and comma(,) separated")
	httpCmd.Flags().StringVarP(&client.Version, "Version", "v", "1", "HTTP version 1 or 2")
	httpCmd.Flags().StringVarP(&client.Body, "body", "b", "", "HTTP body to send")
	httpCmd.Flags().StringVarP(&client.BodyFile, "body_file", "f", "", "File to send as http body")
	httpCmd.Flags().DurationVarP(&client.Timeout, "time_out", "t", 1, "Request timeout in seconds")
	httpCmd.Flags().BoolVar(&client.Keep_alive, "keep_alive", true, "Toggle keep-alive, --keep_alive=[true|false]")
	httpCmd.Flags().BoolVar(&client.Compression, "compression", false, "Toggle compression --compression=[true|false]")
	httpCmd.Flags().BoolVar(&client.Redirect, "redirect", false, "Toggle redirect --redirect=[true|false]")
	rootCmd.AddCommand(httpCmd)
}

func valideHttpArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("Need to define target URI")
	}
	if !helpers.Contains(validMethod, client.Method) {
		return fmt.Errorf("Invalid HTTP methods %s. Valid methods: %v\n", client.Method, validMethod)
	}
	return nil
}

func runHttpCmd(cmd *cobra.Command, args []string) {
	client.Url = args[0]
	runner.Protocol = client
	runner.StatCollector = collector.CreateHttpStatCollector()
	runner.Run()
}
