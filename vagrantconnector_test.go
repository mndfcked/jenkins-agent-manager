package main

import (
	"fmt"
	"testing"
)

func TestSysMemory(t *testing.T) {
	conf, err := NewConfiguration("/etc/jenkins-agent-manager/config.json")
	vacon, err := NewVagrantConnector(conf)
	mem, err := vacon.GetBoxMemory("windows")
	if err != nil || mem != 2147483648 {
		t.Errorf("Fail: %s", err)
	} else {
		fmt.Printf("RAM: %d\n", mem)
	}
}
