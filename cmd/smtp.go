package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/BatikanHyt/netbench/pkg/collector"
	"github.com/BatikanHyt/netbench/pkg/helpers"
	"github.com/BatikanHyt/netbench/pkg/protocols"
	"github.com/spf13/cobra"
)

var smtpClient = protocols.NewSmtpClient()

var smtpCmd = &cobra.Command{
	Use:     "smtp [server_name:port]",
	Run:     runSmtpCmd,
	PreRunE: validateSmtpArgs,
}

func init() {
	smtpCmd.Flags().StringVarP(&smtpClient.From, "from", "f", "", "STMP FROM (required)")
	smtpCmd.Flags().StringArrayVarP(&smtpClient.To, "to", "t", nil, "SMTP to list (required)")
	smtpCmd.Flags().StringVarP(&smtpClient.Subject, "subject", "s", "", "Mail subject (required)")

	smtpCmd.Flags().StringArrayVar(&smtpClient.BCC, "bcc", nil, "SMTP BCC list")
	smtpCmd.Flags().StringArrayVar(&smtpClient.CC, "cc", nil, "SMTP CC list")

	smtpCmd.Flags().BoolVar(&smtpClient.Tls, "tls", false, "Use TLS")
	smtpCmd.Flags().StringVarP(&smtpClient.Auth.Username, "username", "u", "", "Auth username")
	smtpCmd.Flags().StringVarP(&smtpClient.Auth.Password, "password", "p", "", "Auth password")
	smtpCmd.Flags().StringVarP(&smtpClient.Auth.Method, "method", "m", "", "Auth method (CRAM, PLAIN)")
	smtpCmd.Flags().StringVarP(&smtpClient.EmlFile, "eml", "e", "", "Create mail from eml file")
	smtpCmd.Flags().StringVarP(&smtpClient.Body, "body", "b", "", "SMTP text body")
	smtpCmd.Flags().StringVar(&smtpClient.BodyHtml, "bodyhtml", "", "SMTP html body")
	smtpCmd.Flags().StringVar(&smtpClient.BodyFile, "bodyfile", "", "Generate smtp body from file")
	smtpCmd.Flags().StringToStringVarP(&smtpClient.Headers, "headers", "H", nil, "Headers in key=value format and comma(,) separated")
	smtpCmd.Flags().StringArrayVar(&smtpClient.Attachments, "attachment", nil, "List of attachments")

	smtpCmd.MarkFlagRequired("from")
	smtpCmd.MarkFlagRequired("to")
	smtpCmd.MarkFlagRequired("subject")

	rootCmd.AddCommand(smtpCmd)
}

func validateSmtpArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("Needto define STMP server")
	}
	if len(strings.Split(args[0], ":")) != 2 {
		return errors.New("Invalid address format, <ip>:<port>")
	}
	validAuths := []string{"PLAIN", "CRAM"}
	if smtpClient.Auth.Method != "" && !helpers.Contains(validAuths, smtpClient.Auth.Method) {
		return fmt.Errorf("Invalid Auth method %s. Valid auth methods %v\n", smtpClient.Auth.Method, validAuths)
	}
	return nil
}

func runSmtpCmd(cmd *cobra.Command, args []string) {
	smtpClient.Address = args[0]
	runner.Protocol = smtpClient
	runner.StatCollector = collector.CreateSmtpStatCollector()
	runner.Run()
}
