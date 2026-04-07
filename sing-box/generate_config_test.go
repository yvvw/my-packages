package generator_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/sagernet/sing-box/experimental/tools_generate"
)

func TestGenerateConfig(t *testing.T) {
	configName := "config.toml"

	configBytes, err := os.ReadFile(configName)
	if err != nil {
		t.Fatal(err)
	}

	config, err := tools_generate.Parse(configBytes)
	if err != nil {
		t.Fatal(err)
	}

	singboxConfigBytes, err := tools_generate.GenerateSingBoxConfig(configName, config)
	if err != nil {
		t.Fatal(err)
	}

	singboxConfigFile, err := os.Create(config.SingBox.Output)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = singboxConfigFile.Close()
	}()

	fmt.Println(string(singboxConfigBytes))
}
