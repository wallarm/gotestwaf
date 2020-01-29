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
}

func LoadConfig(configFile string) Config {
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	config := Config{}
	err = yaml.Unmarshal(yamlFile, &config)

	return config

}
