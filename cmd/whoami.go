/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"suprsend-cli/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// whoamiCmd represents the whoami command
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		mkey := viper.GetString("management_key")
		log.Debug("SuprSend Key: ", mkey)

		type whoamiResponse struct {
			Org        string `json:"org"`
			Org_id     int    `json:"org_id"`
			Token_name string `json:"token_name"`
		}

		response := []whoamiResponse{
			{Org: "Dhan", Org_id: 666, Token_name: "test"},
			{Org: "Teachmint", Org_id: 999, Token_name: "demo"},
		}
		// response := whoamiResponse{Org: "Dhan", Org_id: 666, Token_name: "test"}
		outputType, _ = cmd.Flags().GetString("output")
		util.OutputData(response, outputType)
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// whoamiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// whoamiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
