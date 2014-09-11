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
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
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
	Index *VagrantIndex
	Boxes *[]Box
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
	vboxes, err := parseBoxes()
	if err != nil {
		return nil, err
	}

	return &VagrantConnector{vindex, vboxes}, nil
}

type Box struct {
	CreatedAt int64
	Name      string
	Provider  string
	Version   int
}

func appendBox(boxes []Box, data ...Box) []Box {
	currLen := len(boxes)
	newLen := currLen + len(data)

	if newLen > cap(boxes) {
		newBoxes := make([]Box, (newLen+1)*2)
		copy(newBoxes, boxes)
		boxes = newBoxes
	}
	boxes = boxes[0:newLen]
	copy(boxes[currLen:newLen], data)
	return boxes
}

func parseBoxes() (*[]Box, error) {
	cmd := exec.Command("vagrant", "box", "list", "--machine-readable")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// we have to ignore errors here because some bug set exit code to 1 even though
	// the command was successfully executed
	//TODO: Revisit later when there was a fix
	_ = cmd.Start()

	scanner := bufio.NewScanner(bytes.NewReader(out))
	var boxes []Box
	boxes = make([]Box, 0)
	var box Box
	for scanner.Scan() {
		str := scanner.Text()
		strpl := strings.Split(str, ",")

		switch strpl[2] {
		case "box-name":
			box = *new(Box)
			box.Name = strpl[3]
		case "box-provider":
			box.Provider = strpl[3]
		case "box-version":
			strV, err := strconv.ParseInt(strpl[3], 0, 0)
			if err != nil {
				return nil, err
			}
			box.Version = int(strV)
			intStamp, err := strconv.ParseInt(strpl[0], 0, 64)
			if err != nil {
				return nil, err
			}
			box.CreatedAt = intStamp
			boxes = appendBox(boxes, box)
		}
	}
	return &boxes, nil
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

func spinUpExec(box string, workingDir string) {
	//	defer waitGrp.Done()
	comm := exec.Command("vagrant", "up")
	comm.Dir = workingDir
	var errOut bytes.Buffer
	var stdOut bytes.Buffer
	comm.Stderr = &errOut
	comm.Stdout = &stdOut
	if err := comm.Start(); err != nil {
		log.Fatalf("[VC] ERROR: While starting command %+v\nERROR: %s\n%s\n", comm, err.Error(), errOut)
	}
	log.Printf("\n%s\n", stdOut)
	if err := comm.Wait(); err != nil {
		log.Fatalf("[VC] ERROR: While running command %+v\nERROR: %s\n%s\n", comm, err.Error(), errOut)
	}
}

func vagrantfileExists(path string) bool {
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatalf("[VS] ERROR: reading directory\nERROR: %s\n", err.Error())
		return false
	}

	for _, e := range entries {
		if !e.IsDir() {
			log.Println("Is no directory")
			log.Printf("Name: %s\n", e.Name())
			if e.Name() == "Vagrantfile" {
				return true
			}
		}
	}
	return false
}

func getBox(label string) (string, error) {
	//TODO: Implement ErrBoxNotFound, supported labels
	return "jenkins-slave-win7", nil
}

func (vc *VagrantConnector) SpinUpNew(label string, workingPath string) error {
	log.Printf("[VC] Trying to start a vagrant machine for the label %s\n", label)
	box, err := getBox(label)
	if err != nil {
		return err
	}

	if !vagrantfileExists(workingPath) {
		initCmd := exec.Command("vagrant", "init", "--force", box)
		initCmd.Dir = workingPath
		var errOut bytes.Buffer
		var out bytes.Buffer
		initCmd.Stderr = &errOut
		initCmd.Stdout = &out

		fmt.Printf("[VC]: Initializing vagrant enviroment at %s with box %s \n", workingPath, box)
		if err := initCmd.Start(); err != nil {
			log.Fatalf("[VC]: ERROR: Can't start command %+v\n", initCmd)
			return err
		}
		log.Printf("[VC]: Command %+v stated, waiting to finish...\n", initCmd)
		if err := initCmd.Wait(); err != nil {
			log.Printf("[VC]: ERROR: Can't spin up box %s at %s\n", box, workingPath)
			log.Fatalf("\nERROR: %s\nOUTPUT: %s\n", errOut.String(), out.String())
			return err
		}
	}

	//	waitGrp := new(sync.WaitGroup)
	//	waitGrp.Add(1)
	fmt.Printf("[VC]: Waiting for spin up to complete, this may take a while\n")
	//	go spinUpExec(box, workingPath, waitGrp)
	//	waitGrp.Wait()
	spinUpExec(box, workingPath)
	return nil
}

func (vc *VagrantConnector) GetBoxMemory() int64 {
	//TODO: Implement
	return 2097152
}
