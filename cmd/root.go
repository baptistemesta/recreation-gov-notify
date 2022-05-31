/*
Copyright © 2022 Kyle Chadha @kylechadha
*/
package cmd

import (
	"fmt"
	"github.com/kylechadha/recreation-gov-notify/notify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"time"
)

var cfgFile string
var campgroundIds string
var dateFrom string
var dateTo string
var allowPartial bool
var phone string
var email string
var pollingInterval string
var debug bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rgn",
	Short: "Get notified when a campsite becomes available on recreation.gov",
	Long: `Get notified when a campsite becomes available on recreation.gov!

This application will periodically poll the recreation.gov API to check whether
the campground you're interested in has any sites available for the date range
you select. That way you can grab up any last minute cancelations.

Get notified by SMS or email. Maybe other stuff in the future, too!`,
	Run: func(cmd *cobra.Command, args []string) {
		runNotify(unmarshallConfig())
	},
}

func unmarshallConfig() *notify.Config {
	config := &notify.Config{}
	err := viper.Unmarshal(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to read configuration:", viper.ConfigFileUsed(), err)
		return nil
	}
	if config.PollInterval == 0 {
		config.PollInterval = 30 * 1000000000
	}
	if config.SMSFrom == "" {
		config.SMSFrom = config.SMSTo
	}
	if config.EmailFrom == "" {
		config.EmailFrom = config.EmailTo
	}
	if dateFrom != "" {
		config.Availabilities.From = dateFrom
	}
	if dateTo != "" {
		config.Availabilities.To = dateTo
	}
	if phone != "" {
		config.SMSTo = phone
	}
	if email != "" {
		config.EmailTo = email
	}
	if pollingInterval != "" {
		config.PollInterval, _ = time.ParseDuration(pollingInterval)
	}
	if campgroundIds != "" {
		config.Availabilities.CampgroundIDs = campgroundIds
	}
	if debug {
		config.Debug = true
	}
	if allowPartial {
		config.Availabilities.Partial = true
	}
	return config
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rgn.yaml)")

	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "activate debug logs")
	rootCmd.Flags().StringVarP(&pollingInterval, "pollingInterval", "r", "", "polling interval, e.g. 30s")
	rootCmd.Flags().StringVarP(&campgroundIds, "campgroundIds", "c", "", "Search a comma separated list of campgrounds")
	rootCmd.Flags().StringVarP(&dateFrom, "dateFrom", "f", "", "date from, format MM-DD-YYYY")
	rootCmd.Flags().StringVarP(&dateTo, "dateTo", "t", "", "date to, format MM-DD-YYYY")
	rootCmd.Flags().BoolVar(&allowPartial, "allowPartial", false, "Allow to have a partial search")
	rootCmd.Flags().StringVar(&phone, "phone", "", "Send notification to that sms number")
	rootCmd.Flags().StringVar(&email, "email", "", "Send notification to that email")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".rgn" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".rgn")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
