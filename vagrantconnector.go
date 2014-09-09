/*
 *
 * Copyright [2014] [Jörn Domnik]
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"
)

var (
	// ErrNoVagrant indicates that no Vagrant installation was found on the system
	ErrNoVagrant = errors.New("No Vagrant installation found")
	// ErrNoMachines indicates that no Vagrant machines where created, for now
	ErrNoMachines = errors.New("No machines found")
)

type vagrantBox struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Version  string `json:"version"`
}

type vagrantExtraData struct {
	Box vagrantBox `json:"box"`
}

type Machine struct {
	LocalDataPath   string           `json:"local_data_path"`
	Name            string           `json:"name"`
	Provider        string           `json:"provider"`
	State           string           `json:"state"`
	VagrantfileName string           `json:"vagrantfile_name"`
	VagrantfilePath string           `json:"vagrantfile_path"`
	UpdatedAt       string           `json:"updated_at"`
	ExtraData       vagrantExtraData `json:"extra_data"`
}

type VagrantIndex struct {
	Version  int                `json:"version"`
	Machines map[string]Machine `json:"machines"`
}

type VagrantConnector struct {
	Index VagrantIndex
}

func NewVagrantConnector() (*VagrantConnector, error) {
	userDir, err := usrDir()
	if err != nil {
		return nil, ErrNoVagrant
	}

	vindex, err := loadVagrantIndex(userDir + "/.vagrant.d/data/machine-index/index")
	if err != nil {
		log.Println("No machine index found, it seems no vargrant boxes have been started.\nCreating empty index.")
		vindex = new(VagrantIndex)
		vindex.Version = 1
		vindex.Machines = make(map[string]Machine)
	}

	return &VagrantConnector{*vindex}, nil
}

func usrDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
}

func loadVagrantIndex(path string) (*VagrantIndex, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var v VagrantIndex
	err = json.Unmarshal(file, &v)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func (vc *VagrantConnector) Print() {
	m := vc.Index
	fmt.Printf("Vagrant Version: %d\n\n", m.Version)
	for k, v := range m.Machines {
		fmt.Printf("Key: %s\n", k)
		fmt.Printf("Name: %s\n", v.Name)
		fmt.Printf("local_data_path: %s\n", v.LocalDataPath)
		fmt.Printf("provider: t%s\n", v.Provider)
		fmt.Printf("state: %s\n", v.State)
		fmt.Printf("vagrantfile name: %s\n", v.VagrantfileName)
		fmt.Printf("vagrantfile path: %s\n", v.VagrantfilePath)
		fmt.Printf("updated at: %s\n", v.UpdatedAt)
		fmt.Printf("extra data: ")
		fmt.Println(v.ExtraData, "\n")
	}
}

func (vc *VagrantConnector) GetVmCount() int {
	return len(vc.Index.Machines)
}

func spinUpExec(cmd string, workingDir string, waitGrp *sync.WaitGroup) {
	comm := exec.Command(cmd)
	comm.Dir = workingDir
	out, err := comm.Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out)
	waitGrp.Done()
}

func (vc *VagrantConnector) SpinUpNew(count int, path string) (int, error) {
	workingPath := filepath.Dir(path)
	box := filepath.Base(path)

	initCmdStr := "vagrant init " + path
	cmd := "vagrant up"

	initCmd := exec.Command(initCmdStr)
	initCmd.Dir = workingPath
	fmt.Printf("[VC]: Initializing vagrant enviroment at %s with box %s \n", workingPath, box)
	out, err := initCmd.CombinedOutput()
	fmt.Printf("[VC]: Output: %s", out)
	err = initCmd.Wait()
	if err != nil {
		log.Printf("[VC]: ERROR: Can't spin up box %s at %s\n", box, workingPath)
		panic(err)
	}

	waitGrp := new(sync.WaitGroup)
	waitGrp.Add(count)

	fmt.Printf("Waiting for spin up to complete, this may take a while\n")
	for i := 0; i <= count; i++ {
		go spinUpExec(cmd, workingPath, waitGrp)
	}

	waitGrp.Wait()

	return count, nil
}

func (vc *VagrantConnector) GetBoxMemory() int64 {
	//TODO: Implement
	return 2097152
}
