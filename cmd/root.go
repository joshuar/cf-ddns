/*
Copyright © 2021 Joshua Rich <joshua.rich@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"net/http"
	"os"
	"time"

	"github.com/joshuar/cf-ddns/internal/cloudflare"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var (
	cfgFile     string
	debugFlag   bool
	profileFlag bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cf-ddns",
	Short: "Update your Cloudflare DNS as your machine's IP address changes",
	Long:  `A Dynamic DNS (DDNS) client for Linux, for domains managed by Cloudflare.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if debugFlag {
			log.SetLevel(log.DebugLevel)
			log.Debug("Debug logging enabled")
		}
		if profileFlag {
			go func() {
				log.Info(http.ListenAndServe("localhost:6060", nil))
			}()
			log.Info("Profiling is enabled and available at localhost:6060")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		cfAccount := cloudflare.GetAccountDetails()
		cfAccount.CheckForUpdates()

		ticker := time.NewTicker(getIntervalFromConfig())
		done := make(chan bool)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				log.Debug("Checking for external IP update...")
				cfAccount.CheckForUpdates()
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "c", "config file (default is $HOME/.cf-ddns.yaml)")
	rootCmd.Flags().BoolVarP(&debugFlag, "debug", "d", false, "debug output")
	rootCmd.Flags().BoolVarP(&profileFlag, "profile", "p", false, "enable profiling")
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

		// Search config in home directory with name ".cf-ddns" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cf-ddns")
	}

	// viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Infof("Using config file: %s", viper.ConfigFileUsed())
	} else {
		log.Fatalf("Error reading config file %s: %v", viper.ConfigFileUsed(), err)
	}
}

func getIntervalFromConfig() time.Duration {
	intervalFromConfig := viper.GetString("interval")
	specifiedInterval, err := time.ParseDuration(intervalFromConfig)
	if err != nil {
		log.Warnf("Couldn't understand interval %s, using default of 1h", intervalFromConfig)
		defaultInterval, _ := time.ParseDuration("1h")
		return defaultInterval
	}
	return specifiedInterval
}
