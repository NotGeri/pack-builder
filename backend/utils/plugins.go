package utils

import (
	"archive/zip"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
)

type PluginConfig struct {
	Name    string   `yaml:"name"`
	Depends []string `yaml:"depend"`
}

// ParsePluginYaml attempts to parse a plugin JAR's plugin.yml
// as a YAML document
func ParsePluginYaml(filePath string) (pluginYaml PluginConfig, err error) {

	// Open the ZIP archive
	zipReader, err := zip.OpenReader(filePath)
	if err != nil {
		return
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		if f.Name == "plugin.yml" {

			// Open the file from the JAR
			var rc io.ReadCloser
			rc, err = f.Open()
			if err != nil {
				return
			}
			defer rc.Close()

			// Read all the data
			var bytes []byte
			bytes, err = io.ReadAll(rc)
			if err != nil {
				return
			}

			// Read the data as a YAML document into our struct
			if err = yaml.Unmarshal(bytes, &pluginYaml); err != nil {
				return
			}

			return
		}
	}

	err = fmt.Errorf("no plugin.yml found")
	return
}
