package db

type Info struct {
	Payload            string
	Encoder            string
	Placeholder        string
	Set                string
	Case               string
	ResponseStatusCode int
	Reason             string
	Type               string
}

type Case struct {
	Payloads       []string `yaml:"payload"`
	Encoders       []string `yaml:"encoder"`
	Placeholders   []string `yaml:"placeholder"`
	Type           string   `yaml:"type"`
	Set            string
	Name           string
	IsTruePositive bool
}
