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
	"os/user"
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

type vagrantMachine struct {
	LocalDataPath   string           `json:"local_data_path"`
	Name            string           `json:"name"`
	Provider        string           `json:"provider"`
	State           string           `json:"state"`
	VagrantfileName string           `json:"vagrantfile_name"`
	VagrantfilePath string           `json:"vagrantfile_path"`
	UpdatedAt       string           `json:"updated_at"`
	ExtraData       vagrantExtraData `json:"extra_data"`
}

type vagrantIndex struct {
	Version  int                       `json:"version"`
	Machines map[string]vagrantMachine `json:"machines"`
}

type VagrantConnector struct {
	VagrantIndex vagrantIndex
}

func NewVagrantConnector() (*VagrantConnector, error) {
	userDir, err := usrDir()
	if err != nil {
		return nil, ErrNoVagrant
	}

	vc, err := loadVagrantIndex(userDir + "/.vagrant.d/data/machine-index/index")

	if err != nil {
		return nil, err
	}

	return &VagrantConnector{*vc}, nil
}

func usrDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
}

func loadVagrantIndex(path string) (*vagrantIndex, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var v vagrantIndex
	err = json.Unmarshal(file, &v)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func (vc *VagrantConnector) Print() {
	m := vc.VagrantIndex
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
	return len(vc.VagrantIndex.Machines)
}

func (vc *VagrantConnector) SpinUpNew(count int) (int, error) {
	//TODO: spin up vagrant boxes
	return 1, nil
}

func (vc *VagrantConnector) GetBoxMemory() int64 {
	//TODO: Implement
	return 2097152
}
