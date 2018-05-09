package cmd

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string
var verbose bool
var uaiFile string
var samplerName string
var randomSeed int64

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grample",
	Short: "(Probalistic) Grpahical Model Sampling Methods",
	Long: `grample provides sampling-based inference for PGM's.
Amoung other features:

  - The ability to read UAI PGM files (for models and evidence)
  - A Gibbs sampler
  - An experimental version of an Adaptive Gibbs sampler
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("grample\n")
		fmt.Printf("Verbose:  %v\n", verbose)
		fmt.Printf("Model:    %s\n", uaiFile)
		fmt.Printf("Sampler:  %s\n", samplerName)
		fmt.Printf("Rnd Seed: %d\n", randomSeed)

		rand.Seed(randomSeed)

		// TODO: actual work
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.grample.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging (default is much more parsimonious)")

	rootCmd.PersistentFlags().StringVarP(&uaiFile, "model", "m", "", "UAI model file to read")
	rootCmd.PersistentFlags().StringVarP(&samplerName, "sampler", "s", "", "Name of sampler to use")
	rootCmd.PersistentFlags().Int64VarP(&randomSeed, "seed", "r", 1, "Random seed to use")

	rootCmd.MarkPersistentFlagRequired("model")
	rootCmd.MarkPersistentFlagRequired("sampler")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
