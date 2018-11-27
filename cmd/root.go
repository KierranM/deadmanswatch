package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "deadmanswatch",
	Short: "Forwards Prometheus DeadManSwitch alerts to CloudWatch metrics",
	Long: `Listens for DeadMansSwitch alerts for AlertManager and forwards
them as metrcs to CloudWatch so that you can create alarms from them.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		logrus.Warnf("Failed to display help: %v", err)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
}
