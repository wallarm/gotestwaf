package config

import (
	"io/ioutil"
	"log"
	"net/http"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Headers                map[string]string `yaml:"headers"`
	Proxy                  string            `yaml:"proxy"`
	CertificateCheck       bool              `yaml:"certificatecheck"`
	MaxIddleConnections    int               `yaml:"threads"`
	IddleConnectionTimeout int               `yaml:"threadTimeout"`
	TestcasesFolder        string            `yaml:"testcasesFolder"`
	BlockStatusCode        int               `yaml:"blockStatusCode"`
	BlockRegExp            string            `yaml:"blockRegExp"`
	PassStatusCode         int               `yaml:"passStatusCode"`
	PassRegExp             string            `yaml:"passRegExp"`
	ReportFile             string            `yaml:"reportFile"`
	PayloadsExportFile     string            `yaml:"payloadsExportFile"`
	NonBlockedAsPassed     bool              `yaml:"nonBlockedAsPassed"`
	Cookies                []*http.Cookie    ``
	FollowCookies          bool              `yaml:"followCookies"`
	MaxRedirects           int               `yaml:"maxRedirects"`
	SendingDelay           int               `yaml:"sendingDelay"`
	RandomDelay            int               `yaml:"randomDelay"`
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
