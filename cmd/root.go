package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grample",
	Short: "(Probalistic) Grpahical Model Sampling Methods",
	Long: `grample provides sampling-based inference for PGM's.
Amoung other features:

  - A Gibbs sampler
  - The ability to read UAI PGM files (for models and evidence)
  - An experimental version of an Adaptive Gibbs sampler
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// TODO: put individual commands in their file with actual testing
	// TODO: actual flags we need
	// TODO: don't forget to bind them to viper (and use viper to read them)
	// TODO: set default values for flags
	// TODO: unmarshal config from viper into a single struct
	// TODO: cmd - config, which prints out config (do this from config struct above)
	// TODO: cmd - read, which loads a file, verifies the model, and output a descrip
	// TODO: REAL WORK
	// TODO: update readme with flags, giving command line, config file, and env var names (and default values)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.grample.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("verbose", "v", false, "Verbose logging (default is much more parsimonious)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".grample" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".grample")
	}

	viper.SetEnvPrefix("GRAMPLE")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
