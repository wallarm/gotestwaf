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

type Case struct {
	Payloads       []string `yaml:"payload"`
	Encoders       []string `yaml:"encoder"`
	Placeholders   []string `yaml:"placeholder"`
	Type           string   `default:"unknown" yaml:"type"`
	Set            string
	Name           string
	IsTruePositive bool
}
