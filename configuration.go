package main

type Configuration struct {
	JenkinsUrl   string
	JenkinsPort  string
	ListenerPort string
	MaxVms       int
	BoxPath      string
}

func NewConfiguration(ju string, jp string, lp string, mv int, boxpath string) *Configuration {
	return &Configuration{ju, jp, lp, mv, boxpath}
}
