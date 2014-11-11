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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

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
	for _, box := range *vBoxes {
		box.Print()
	}

	// Create a new vagrant connector and return it
	return &VagrantConnector{vIndex, vBoxes, conf}, nil
}

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

func (vc *VagrantConnector) StartMachineFor(label string, workingPath string) error {
	log.Printf("[VagrantConnector] Trying to start a vagrant machine for the label %s\n", label)
	boxName, err := vc.GetBoxNameFor(label)

	if err != nil {
		log.Printf("[VagrantConnector]: ERROR while retriving box name for label %s. Error: %s\n", label, err)
		return err
	}

	vagrantfilePath := filepath.Join(workingPath, boxName, "Vagrantfile")
	vagrantfileDirPath := filepath.Dir(vagrantfilePath)

	if !govagrant.VagrantfileExists(vagrantfilePath) {
		log.Printf("[VagrantConnector]: Vagrantfile not found, creating new in path %s\n", vagrantfilePath)
		if err := os.MkdirAll(vagrantfileDirPath, 0755); err != nil {
			log.Printf("[VagrantConnector]: ERROR: Can't create the working directory for label %s on path %s. Error message: %s\n", label, workingPath, err.Error())
			return err
		}

		govagrant.Init(vagrantfilePath, boxName)
	}
	fmt.Printf("[VagrantConnector]: Waiting for spin up to complete, this may take a while\n")

	govagrant.Up(vagrantfilePath)

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
	log.Printf("[VagrantConnector] Received request to destroy a box with the label %s.\n", label)

	box, err := vc.GetBoxNameFor(label)
	if err != nil {
		log.Printf("[VagrantConnector] Cannot destroy a machine for label %s. No box found for that label. Error: %s.\n", label, err.Error())
		return err
	}

	vagrantfilePath := filepath.Join(workingDir, box, "Vagrantfile")
	machines, err := govagrant.Status(vagrantfilePath)
	if err != nil {
		log.Printf("[VagrantConnector] Error while getting the vagrant status for %s. Error: %s.\n", vagrantfilePath, err.Error())
		return err
	}

	for _, m := range *machines {
		if m.State == "running" {
			log.Printf("[VagrantConnector] Found machine %s with running state. Trying to destroy it\n", m.Name)
			return govagrant.Destroy(vagrantfilePath)
		}
	}

	return nil
}

func (vc *VagrantConnector) GetRunningMachineCount() (int, error) {
	path := vc.Config.WorkingDirPath
	count := 0
	dirs, err := ioutil.ReadDir(path)
	if err != nil {
		log.Printf("[VagrantConnector] ERROR while reading running vagrant machines in %s. Error: %s\n", path, err)
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			log.Printf("[VagrantConnector] Checking %s for Vagrantfile\n", path)
			machines, err := govagrant.Status(filepath.Join(path, dir.Name(), "Vagrantfile"))

			if err != nil && err != govagrant.ErrVagrantfileNotFound {
				log.Printf("[VagrantConnector]: Error while getting vagrant status in path %s. Error: %s\n", vc.Config.WorkingDirPath, err.Error())
				return 0, err
			} else if err != nil && err == govagrant.ErrVagrantfileNotFound {
				continue
			}

			for _, m := range *machines {
				if m.State == "running" {
					count++
				}
			}
		}
	}

	return count, nil
}
