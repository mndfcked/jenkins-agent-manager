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
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/docker/docker/pkg/units"
	"github.com/mndfcked/govagrant"
)

// ErrorBoxNotFound indicats that no box was configured for the defined label
var ErrBoxNotFound = errors.New("No boxes for the specified lable found")

type VagrantConnector struct {
	Index  *govagrant.VagrantMachineIndex
	Boxes  *[]govagrant.VagrantBox
	Config *Configuration
}

func NewVagrantConnector(conf *Configuration) (*VagrantConnector, error) {
	// Parse the vagrant machines index and save them
	vIndex, err := govagrant.GetMachineIndex()
	if err != nil {
		log.Println("[VC]: No machine index found, it seems no vargrant boxes have been started. Creating empty index.")
		vIndex = new(govagrant.VagrantMachineIndex)
		vIndex.Version = 1
		vIndex.Machines = make(map[string]govagrant.VagrantMachine)
	}
	// Parse all current vagrant boxes and save them
	vBoxes, err := govagrant.ParseBoxes()
	if err != nil {
		return nil, err
	}

	// Create a new vagrant connector and return it
	return &VagrantConnector{vIndex, vBoxes, conf}, nil
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

//TODO: Fixup
func (vc *VagrantConnector) GetBoxNameFor(label string) (string, error) {
	boxes := vc.Config.Boxes
	for _, box := range boxes {
		for _, boxLabel := range box.Labels {
			if boxLabel == label {
				return box.Name, nil
			}
		}
	}
	return "", fmt.Errorf("No box for the label %s configured.", label)
}

//TODO: Fixup
func (vc *VagrantConnector) StartMachineFor(label string, workingPath string) error {
	log.Printf("[VC] Trying to start a vagrant machine for the label %s\n", label)
	boxName, err := vc.GetBoxNameFor(label)
	if err != nil {
		return err
	}

	boxPath := workingPath + "/" + boxName
	if err := os.MkdirAll(boxPath, 0755); err != nil {
		log.Printf("[VagrantConnector]: ERROR: Can't create the working directory for label %s on path %s. Error message: %s", label, workingPath, err.Error())
		return err
	}
	initCmd := exec.Command("vagrant", "init", "--force", boxName)
	initCmd.Dir = boxPath
	var errOut bytes.Buffer
	var out bytes.Buffer
	initCmd.Stderr = &errOut
	initCmd.Stdout = &out

	fmt.Printf("[VagrantConnector]: Initializing vagrant enviroment at %s with box %s \n", boxPath, boxName)
	if err := initCmd.Start(); err != nil {
		log.Printf("[VagrantConnector]: ERROR: Can't start command %+v\n", initCmd)
		return err
	}
	log.Printf("[VagrantConnector]: Command %+v stated, waiting to finish...\n", initCmd)
	if err := initCmd.Wait(); err != nil {
		log.Printf("[VC]: ERROR: Can't spin up box %s at %s\n", boxName, boxPath)
		log.Printf("\nERROR: %s\nOUTPUT: %s\n", errOut.String(), out.String())
		return err

	}

	fmt.Printf("[VagrantConnector]: Waiting for spin up to complete, this may take a while\n")
	spinUpExec(boxName, boxPath)

	return nil
}

func (vc *VagrantConnector) GetBoxMemoryFor(label string) (int64, error) {
	for _, box := range vc.Config.Boxes {
		for _, l := range box.Labels {
			if l == label {
				return units.RAMInBytes(box.Memory)
			}
		}
	}
	return -1, ErrBoxNotFound
}

func (vc *VagrantConnector) DestroyMachineFor(label string, workingDir string) error {
	box, err := vc.GetBoxNameFor(label)
	if err != nil {
		log.Printf("[VagrantConnector]: Cannot destroy a machine for label %s. No box found for that label. Error: %s\n", label, err.Error())
		return err
	}

	vi, err := govagrant.GetMachineIndex()
	if err != nil {
		log.Printf("[VagrantConnector]: Error while loading the vagrant index. Error: %s\n", err.Error())
		return err
	}

	for _, m := range vi.Machines {
		if m.Name == box && m.State == "running" {
			return destroyBox(box, workingDir+box)
		}
	}

	return nil
}

func destroyBox(name string, workingDir string) error {
	cmd := exec.Command("vagrant", "destroy")

	boxPath := workingDir + name
	cmd.Dir = boxPath

	var errOut bytes.Buffer
	var stdOut bytes.Buffer
	cmd.Stderr = &errOut
	cmd.Stdout = &stdOut

	if err := cmd.Start(); err != nil {
		log.Printf("[VagrantConnector]: Error while staring the vagrant destory command in path %s. Error: %s\n", boxPath, err.Error())
		return err
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("[VagrantConnector]: Error while running the vagrant destroy command in path %s. Error: %s\n", boxPath, err.Error())
	}

	return nil
}

func (vc *VagrantConnector) GetRunningMachineCount() (int, error) {
	machines, err := govagrant.Status(vc.Config.WorkingDirPath)
	if err != nil {
		log.Printf("[VagrantConnector]: Error while getting vagrant status in path %s. Error: %s\n", vc.Config.WorkingDirPath, err.Error())
		return -1, err
	}
	runningMachines := 0
	for _, m := range *machines {
		if m.State == "running" {
			runningMachines++
		}
	}

	return runningMachines, nil
}
