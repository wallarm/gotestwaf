package db

type Info struct {
	Payload            string
	Encoder            string
	Placeholder        string
	Set                string
	Case               string
	ResponseStatusCode int
	AdditionalInfo     []string
	Type               string
}

type yamlConfig struct {
	Payloads     []string `yaml:"payload"`
	Encoders     []string `yaml:"encoder"`
	Placeholders []any    `yaml:"placeholder"` // array of string or map[string]any
	Type         string   `default:"unknown" yaml:"type"`
}

type Case struct {
	Payloads     []string
	Encoders     []string
	Placeholders []*Placeholder
	Type         string

	Set            string
	Name           string
	IsTruePositive bool
}

type Placeholder struct {
	Name   string
	Config any
}
