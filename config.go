package ktlogging

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v2"
)

// for json/yaml config file parsing - this is the entries in /loggers path
type LoggerConfigModel struct {
	Name         string   `json:"name" yaml:"name"`
	Level        string   `json:"level" yaml:"level"`
	HandlerNames []string `json:"handlers" yaml:"handlers"`
}

// for json/yaml config file parsing - this is the entries in /handlers path
type HandlerConfigModel struct {
	Level       string   `json:"level" yaml:"level"`
	Encoding    string   `json:"encoding" yaml:"encoding"`
	OutputPaths []string `json:"outputPaths" yaml:"outputPaths"`
}

// for json/yaml config file parsing - this is root level object
type ConfigModel struct {
	Loggers  map[string]LoggerConfigModel  `json:"loggers" yaml:"loggers"`
	Handlers map[string]HandlerConfigModel `json:"handlers" yaml:"handlers"`
}

// parsing the ConfigModel from the given file path - which must be either JSON or Yaml file
func parseFromJsonOrYaml(cfgPath string) (ConfigModel, error) {

	// let's read config content
	cfgFile, openErr := os.Open(cfgPath)
	if openErr != nil {
		return ConfigModel{}, fmt.Errorf("failed to open log config! error was: %v", openErr)
	}
	defer cfgFile.Close()
	byteValue, _ := ioutil.ReadAll(cfgFile)

	// lets (try to) parse into our config struct!
	var config ConfigModel

	// json or yaml?
	extension := path.Ext(strings.ToLower(cfgPath))
	switch extension {
	case ".yaml":
		{
			if err := yaml.Unmarshal(byteValue, &config); err != nil {
				return ConfigModel{}, fmt.Errorf("failed to parse log config! error was: %v", err)
			}
		}
	case ".json":
		{
			if err := json.Unmarshal(byteValue, &config); err != nil {
				return ConfigModel{}, fmt.Errorf("failed to parse log config! error was: %v", err)
			}
		}
	default:
		{
			return ConfigModel{}, fmt.Errorf("unknown config file extension '%v'! Only .json or .yaml is supported!", extension)
		}
	}

	return config, nil
}
