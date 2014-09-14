package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type Configuration struct {
	JenkinsApiUrl    string    `json:"jenkins_api_url"`
	JenkinsApiSecret string    `json:"jenkins_api_secret"`
	ListenerPort     string    `json:"listener_port"`
	MaxVms           int       `json:"max_vm_count"`
	WorkingDirPath   string    `json:"working_dir_path"`
	Boxes            []confBox `json:"boxes"`
}

type confBox struct {
	Name   string   `json:"name"`
	Labels []string `json:"labels"`
}

func NewConfiguration(confFile string) (*Configuration, error) {
	c, err := parseConfFile(confFile)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func parseConfFile(path string) (*Configuration, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("[CONF]: Error while trying to read configuration file at %s.\nError: %s\n", path, err.Error())
		return nil, err
	}

	var c Configuration
	err = json.Unmarshal(file, &c)
	if err != nil {
		log.Fatalf("[CONF]: Error while parsing the configuration file.\nError: %s\n", err.Error())
		return nil, err
	}

	return &c, nil
}
