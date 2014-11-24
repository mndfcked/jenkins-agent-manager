//
//Copyright [2014] [Jörn Domnik]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/units"
	"github.com/mndfcked/govagrant"
)

// BoxNotFoundError tells the caller that no vagrant box for the given label was configured in the configuration file
type BoxNotFoundError struct {
	Label string
}

func (e *BoxNotFoundError) Error() string {
	return fmt.Sprintf("No Box for label %s configured", e.Label)
}

// VagrantConnector struct holds references to vagrants machine index, vagrants boxs index and the configuration module
type VagrantConnector struct {
	Index  *govagrant.VagrantMachineIndex
	Boxes  []govagrant.VagrantBox
	Config *Configuration
}

// NewVagrantConnector loads the vagrant machine index, vagrant boxes index and creates a new VagrantConnector struct which holds references to these two and the configuration module
func NewVagrantConnector(conf *Configuration) (*VagrantConnector, error) {
	// Parse the vagrant machines index and save them
	log.Println("[VagrantConnector] Loading machine indes...")

	vIndex, err := govagrant.GetMachineIndex()
	if err != nil {
		log.Println("[VagrantConnector] No machine index found, it seems no vargrant boxes have been started. Creating empty index.")
		vIndex = new(govagrant.VagrantMachineIndex)
		vIndex.Version = 1
		vIndex.Machines = make(map[string]govagrant.VagrantMachine)
	}
	log.Println("[VagrantConnector] Successfully loaded the following machines:")
	vIndex.Print()

	// Parse all current vagrant boxes and save them
	log.Println("[VagrantConnector] Loading boxes list...")

	vBoxes, err := govagrant.BoxList()
	if err != nil {
		return nil, err
	}

	log.Println("[VagrantConnector] Successfully oaded the following boxes:")
	for _, box := range vBoxes {
		box.Print()
	}

	// Create a new vagrant connector and return it
	return &VagrantConnector{vIndex, vBoxes, conf}, nil
}

func (vc *VagrantConnector) SnapshotMachine(id string, workingPath string) (string, error) {
	vagrantfilePath := filepath.Join(workingPath, id)
	snapshotID, err := govagrant.SnapTake(vagrantfilePath)

	if err != nil {
		log.Printf("[VagrantConnector] Error while taking a snapshot of the machine %s in path %s. Error: %s", id, workingPath, err)
		return "", err
	}

	return snapshotID, nil
}

// GetBoxNameFor takes a label as paramter and searches the configuration for a suitable box with the label and returns the name of this box.
func (vc *VagrantConnector) GetBoxNameFor(label string) (string, error) {
	boxes := vc.Config.Boxes

	for _, box := range boxes {
		for _, boxLabel := range box.Labels {
			if boxLabel == label {
				return box.Name, nil
			}
		}
	}

	return "", &BoxNotFoundError{label}
}

// CreateMachineFor takes a label and workingPath as parameter, creates a new vagrantfile in it and starts the machine up. Afert a successfuly start, the ID of the new machine is returned.
func (vc *VagrantConnector) CreateMachineFor(label string, workingPath string) (string, error) {
	log.Printf("[VagrantConnector] Got request for creating a new vagrant machine for the label %s in the path %s.\n", label, workingPath)
	boxName, err := vc.GetBoxNameFor(label)

	if err != nil {
		log.Printf("[VagrantConnector]: ERROR while retriving box name for label %s. Error: %s\n", label, err)
		return "", err
	}

	vagrantfilePath := filepath.Join(workingPath, "Vagrantfile")
	vagrantfileDirPath := filepath.Dir(vagrantfilePath)

	if !govagrant.VagrantfileExists(vagrantfilePath) {
		log.Printf("[VagrantConnector]: Vagrantfile not found, creating new in path %s\n", vagrantfilePath)
		if err := os.MkdirAll(vagrantfileDirPath, 0755); err != nil {
			log.Printf("[VagrantConnector]: ERROR: Can't create the working directory for label %s on path %s. Error message: %s\n", label, workingPath, err.Error())
			return "", err
		}

		govagrant.Init(vagrantfilePath, boxName)
	}
	log.Printf("[VagrantConnector]: Waiting for spin up to complete, this may take a while\n")

	if err := govagrant.Up(vagrantfilePath); err != nil {
		log.Printf("[VagrantConnector] Error while starting vagrant machine in path %s. Error: %s\n", vagrantfilePath, err)
		return "", err
	}

	return filepath.Base(workingPath), nil
}

// StartMachineIn takes a workingPath as parameter and tries to start the containing vagrant machine.
func (vc *VagrantConnector) StartMachineIn(workingPath string) error {
	vagrantfilePath := filepath.Join(workingPath, "Vagrantfile")
	log.Printf("[VagrantConnector] Trying to start a vagrant machine in the path %s\n", vagrantfilePath)

	if !govagrant.VagrantfileExists(vagrantfilePath) {
		log.Printf("[VagrantConnector]: Vagrantfile not found, creating new in path %s\n", vagrantfilePath)
		return govagrant.ErrVagrantfileNotFound
	}

	status, err := govagrant.Status(vagrantfilePath)
	if err != nil {
		log.Printf("[VagrantConnector] Error while retirivng status for vagrant machine in %s. Error: %s\n", vagrantfilePath, err)
		return err
	}

	for _, m := range status {
		if m.State == "Saved" {
			if err := govagrant.Up(vagrantfilePath); err != nil {
				log.Printf("[VagrantConnector] Error while starting vagrant machine in path %s. Error: %s\n", vagrantfilePath, err)
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("Couldn't start machine in path %s.", vagrantfilePath)
}

// GetBoxMemoryFor takes a label as parameter, searches the configuration for a suitable box and returns the system memory configured for this box.
func (vc *VagrantConnector) GetBoxMemoryFor(label string) (int64, error) {
	for _, box := range vc.Config.Boxes {
		for _, l := range box.Labels {
			if l == label {
				return units.RAMInBytes(box.Memory)
			}
		}
	}

	return -1, &BoxNotFoundError{label}
}

// ResetMachineIn resets the vagrant machine located in the passed working directory to the snapshot passed in the second parameter.
func (vc *VagrantConnector) ResetMachineIn(workingDir string, snapshotID string) error {
	log.Printf("[VagrantConnector] Received request to reset the machine in path %s.\n", workingDir)

	vagrantfilePath := filepath.Join(workingDir, "Vagrantfile")
	if err := govagrant.SnapBack(vagrantfilePath, snapshotID); err != nil {
		log.Printf("[VagrantConnector] Error while resetting the machine in path %s to snapshot %s. Error: %s\n",
			workingDir,
			snapshotID,
			err)
		return err
	}

	return nil
}

// DestroyMachineFor takes a working directory path as parameter, tries to destroy the box and returns the state the machine has after the destory command.
func (vc *VagrantConnector) DestroyMachineFor(workingDir string) (string, error) {
	//TODO: Don't destroy. Instead=> Snapshot restoring
	//TODO: Return the correct, non hardcoded, state
	log.Printf("[VagrantConnector] Received request to destroy the machine in path %s.\n", workingDir)

	vagrantfilePath := filepath.Join(workingDir, "Vagrantfile")
	machines, err := govagrant.Status(vagrantfilePath)
	if err != nil {
		log.Printf("[VagrantConnector] Error while getting the vagrant status for %s. Error: %s.\n", vagrantfilePath, err.Error())
		return "", err
	}

	for _, m := range machines {
		if m.State == "running" {
			log.Printf("[VagrantConnector] Found machine %s with running state. Trying to destroy it\n", m.Name)

			if err := govagrant.Destroy(vagrantfilePath); err != nil {
				log.Printf("[VagrantConnector] Error while destroying the machine in path %s. Error: %s\n", workingDir, err)
				return "", err
			}
		}
	}
	state := "destroyed"
	return state, nil
}

// GetRunningMachineCount traverses the configured workingpath base dir and calls vagrant status for every vagrant file it finds.
// Every "running" vagrant machine will be counted and the total amount of "running" machines will be returned.
func (vc *VagrantConnector) GetRunningMachineCount() (int, error) {
	workingPath := vc.Config.WorkingDirPath
	machinesMap := map[string]*govagrant.VagrantMachine{}
	dirs, err := ioutil.ReadDir(workingPath)
	if err != nil {
		log.Printf("[VagrantConnector] ERROR while reading running vagrant machines in %s. Error: %s\n", workingPath, err)
		return -1, err
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			path := filepath.Join(workingPath, dir.Name(), "Vagrantfile")
			log.Printf("[VagrantConnector]\t=> Checking status for %s.\n", path)
			machines, err := govagrant.Status(path)

			if err == govagrant.ErrVagrantfileNotFound {
				log.Printf("[VagrantConnector]\t=> No Vagrantfile in %s. Continuing...\n", path)
				continue
			} else if err != nil {
				log.Printf("[VagrantConnector]: Error while getting vagrant status in path %s. Error: %s\n", vc.Config.WorkingDirPath, err.Error())
				return -1, err
			}

			for _, m := range machines {
				if m.State == "running" {
					machinesMap[dir.Name()] = &m
					log.Printf("[VagrantConnector]\t=> Found a machine with state %s for %s. Counting...\n", m.State, path)
				}
			}
		}
	}

	return len(machinesMap), nil
}
