package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "benchmark",
		Short: "Go Web Framework Benchmarking Suite",
		Long: `A comprehensive benchmarking suite for comparing performance 
across different Go web frameworks including GoFlash, Gin, Fiber, Echo, and Chi.

Features:
- Atomic and deterministic test execution
- Resume capability from failed runs
- YAML-based configuration
- Progress tracking with beautiful output
- Comprehensive result collection and analysis`,
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().StringP("results-dir", "r", "", "results directory (overrides config)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolP("dry-run", "d", false, "dry run mode (don't execute tests)")

	// Bind flags to viper
	viper.BindPFlag("output.results_dir", rootCmd.PersistentFlags().Lookup("results-dir"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("dry_run", rootCmd.PersistentFlags().Lookup("dry-run"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("Error reading config file: %s\n", err)
			os.Exit(1)
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
