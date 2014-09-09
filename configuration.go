package main

type Configuration struct {
	JenkinsUrl     string
	JenkinsPort    string
	ListenerPort   string
	MaxVms         int
	BoxPath        string
	WorkingDirPath string
}

func NewConfiguration(ju string, jp string, lp string, mv int, boxPath string, workingDir string) *Configuration {
	return &Configuration{ju, jp, lp, mv, boxPath, workingDir}
}
