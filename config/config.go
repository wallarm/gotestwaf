package config

import (
	"io/ioutil"
	"log"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Headers                map[string]string `yaml:"headers"`
	Proxy                  string            `yaml:"proxy"`
	CertificateCheck       bool              `yaml:"certificatecheck"`
	MaxIddleConnections    int               `yaml:"threads"`
	IddleConnectionTimeout int               `yaml:"threadTimeout"`
	TestcasesFolder        string            `yaml:"testcasesFolder"`
}

func LoadConfig(configFile string) *Config {
	if yamlFile, err := ioutil.ReadFile(configFile); err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		return nil
	} else {
		config := Config{}
		err = yaml.Unmarshal(yamlFile, &config)
		return &config
	}
}
