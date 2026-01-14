package kt_logging

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

type RollingFileModel struct {
	// File is the file path to write logs to.
	File string `json:"file" yaml:"file"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSizeMb int `json:"maxSizeMb" yaml:"maxSizeMb"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAgeDays int `json:"maxAgeDays" yaml:"maxAgeDays"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int `json:"maxBackups" yaml:"maxBackups"`

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool `json:"compress" yaml:"compress"`
}

// for json/yaml config file parsing - this is the entries in /handlers path
type HandlerConfigModel struct {
	Level       string            `json:"level" yaml:"level"`
	Encoding    string            `json:"encoding" yaml:"encoding"`
	OutputPaths []string          `json:"outputPaths" yaml:"outputPaths"`
	RollingFile *RollingFileModel `json:"rollingFile" yaml:"rollingFile"`
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
