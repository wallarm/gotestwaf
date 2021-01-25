package test

type Test struct {
	Payload     string
	Encoder     string
	Placeholder string
	TestSet     string
	TestCase    string
	StatusCode  int
}

type TestCase struct {
	Payloads     []string `yaml:"payload"`
	Encoders     []string `yaml:"encoder"`
	Placeholders []string `yaml:"placeholder"`
	Set          string
	Name         string
	Type         bool
}
