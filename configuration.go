package main

type Configuration struct {
	JenkinsUrl   string
	JenkinsPort  string
	ListenerPort string
	MaxVms       int
}

func NewConfiguration(ju string, jp string, lp string, mv int) *Configuration {
	return &Configuration{ju, jp, lp, mv}
}
