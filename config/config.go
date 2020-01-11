package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// Config structure for server
type Config struct {
	Heartbeat int    `yaml:"heartbeat_interval"`
	Collect   int    `yaml:"collect_interval"`
	Gateway   string `yaml:"gateway_addr"`
	BaseDir   string `yaml:"base_dir"`
	TLSSwitch bool   `yaml:"tls_switch"`
	PermFile  string `yaml:"tls_perm"`
	KeyFile   string `yaml:"tls_key"`
	CaFile    string `yaml:"tls_ca"`
}

// ParseYamlFile the config file
func ParseYamlFile(filename string, c *Config) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}
