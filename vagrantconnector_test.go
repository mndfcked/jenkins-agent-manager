package main

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/mndfcked/govagrant"
)

func TestGetBoxMemory(t *testing.T) {
	const sysMemBytes = 2147483648
	conf, err := mockConfig()
	vacon := mockVagrantConnector(conf)
	mem, err := vacon.GetBoxMemoryFor("windows")
	if err != nil || mem != sysMemBytes {
		t.Errorf("Fail: %s", err)
	}
}

func TestGetBox(t *testing.T) {
	const boxLabel = "windows"
	conf, err := mockConfig()
	vacon := mockVagrantConnector(conf)
	box, err := vacon.GetBoxNameFor(boxLabel)
	if err != nil || box != "win7-slave" {
		t.Errorf("Fail: %s", err)
	}
}

func mockConfig() (*Configuration, error) {
	var c Configuration
	var configJson = []byte(`{
		  "jenkins_api_url":"http://localhost:8080",
		  "jenkins_api_secret":"",
		  "listener_port":"8888",
		  "max_vm_count":2,
		  "working_dir_path":"/tmp",
		  "boxes":
		  [{
			  "name": "win7-slave",
			  "labels": ["windows"],
			  "memory": "2048MB"
		  	}]
		}`)
	if err := json.Unmarshal(configJson, &c); err != nil {
		log.Fatalf("[TEST VAGRANTCONNECOR]: Error while parsing the test config json string. Reson: %s", err)
		return nil, err
	}
	return &c, nil
}

func mockVagrantConnector(conf *Configuration) *VagrantConnector {
	vagrantIndex := new(govagrant.VagrantMachineIndex)
	vagrantIndex.Version = 1
	vagrantIndex.Machines = make(map[string]govagrant.VagrantMachine)
	var vagrantBoxes []govagrant.VagrantBox
	vagrantBoxes = make([]govagrant.VagrantBox, 1, 1)
	vagrantBoxes[0] = govagrant.VagrantBox{"Test-Box", "Test-Provider", "1.0", 1234}
	return &VagrantConnector{vagrantIndex, &vagrantBoxes, conf}
}
