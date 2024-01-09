package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/aws/aws-sdk-go/aws"
)

const Version = "1.0.1"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of deadmanswatch",
	Long:  `All software has versions. This is DeadMansWatch's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("DeadMansWatch v%s\n", Version)
		fmt.Printf("  AWS SDK Version: v%s", aws.SDKVersion)
	},
}
