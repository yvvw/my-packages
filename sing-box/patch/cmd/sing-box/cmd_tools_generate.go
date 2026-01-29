//go:build with_tools_generate

package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/sagernet/sing-box/experimental/tools_generate"
)

var commandToolsGenerate = &cobra.Command{
	Use:  "generate <config> [<config> ...]",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup
		wg.Add(len(args))
		for _, arg := range args {
			go func(configName string) {
				defer wg.Done()
				if err := toolsGenerate(configName); err != nil {
					fmt.Fprintf(os.Stderr, "generate failed for %s: %v\n", configName, err)
					os.Exit(1)
				}
			}(arg)
		}
		wg.Wait()
	},
}

func init() {
	commandTools.AddCommand(commandToolsGenerate)
}

func toolsGenerate(configName string) error {
	configBytes, err := os.ReadFile(configName)
	if err != nil {
		return err
	}

	config, err := tools_generate.Parse(configBytes)
	if err != nil {
		return err
	}

	singboxConfigBytes, err := tools_generate.GenerateSingBoxConfig(configName, config)
	if err != nil {
		return err
	}

	singboxConfigFile, err := os.Create(config.SingBox.Output)
	if err != nil {
		return err
	}
	defer func() {
		_ = singboxConfigFile.Close()
	}()

	_, err = singboxConfigFile.Write(singboxConfigBytes)
	if err != nil {
		return err
	}

	return nil
}
