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

//TODO: Implement new design
//	- check before every start of a new machine, wheter it there is a free, already created machine available
//	- don't destroy a machine, snapshot and restore instead (=> vagrantconnector, only when already created)

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/gosigar"
)

// A TooManyVmsError tells the caller that the via configuration defined maximum count von machines is exeeded
type TooManyVmsError struct {
	RunningVms   int
	RequestedVms int
}

func (e *TooManyVmsError) Error() string {
	return fmt.Sprintf("Too many VMs are running. Max. allowed VMs are %d requested: %d", e.RunningVms, e.RequestedVms)
}

// A NoFreeMemoryError tells the caller that there is not enough free system memory available to start the new machine
type NoFreeMemoryError struct {
	FreeMemory      uint64
	RequestedMemory uint64
}

func (e *NoFreeMemoryError) Error() string {
	return fmt.Sprintf("Not enought free systen memory. The System has %d Bytes of free system memory, requested where %d Bytes", e.RequestedMemory, e.FreeMemory)
}

// A NoFreeMachineError tells the caller that non of the already created machine is free for a new job.
var NoFreeMachineError = errors.New("no machines found.")

/*
func (e *NoFreeMachineError) Error() string {
	return fmt.Sprintf("no unused vagrant machine for label %s found", e.Label)
}
*/
// Controller struct gives other type to hold reference to it
type Controller struct {
	VagrantConnector *VagrantConnector
	JenkinsConnector *JenkinsConnector
	Config           *Configuration
	Database         *DbHelper
}

// NewController instatiates a new Controller and returns it
func NewController(vc *VagrantConnector, jc *JenkinsConnector, conf *Configuration, dbhelper *DbHelper) (*Controller, error) {
	return &Controller{vc, jc, conf, dbhelper}, nil
}

// StartVms takes a box label as parameter, creates a unique id and starts the machine inside a unique folder. After a successufl start, the new machine will be stored in the database and the id returned to the listener
func (c *Controller) StartVms(label string) (string, error) {
	log.Printf("[Controller] Received request to start a box for label %s.\n", label)

	boxMemory, err := c.VagrantConnector.GetBoxMemoryFor(label)
	if err != nil {
		return "", err
	}

	if err := c.checkMaxVMCount(); err != nil {
		log.Printf("[Controller] Error while checking for max. concurrently allowed machines. Error: %s\n", err)
		return "", err
	}

	if err := checkFreeSysMem(uint64(boxMemory)); err != nil {
		log.Printf("[Controller] Error while checking free system memory for box %s. Error: %s\n", label, err)
		return "", err
	}

	workingPath, err := c.checkUnusedMachineFor(label, c.Config.WorkingDirPath)
	var id string
	if err == NoFreeMachineError {
		log.Println("[Controller] No unused Machine found, trying to create a new one.")
		id, err = c.VagrantConnector.CreateMachineFor(label, workingPath)
		if err != nil {
			log.Printf("[Controller] Error while creating a new vagrant machine for %s label in %s. Error: %s\n", label, workingPath, err)
			return "", err
		}
		workingPath = filepath.Join(c.Config.WorkingDirPath, id)
		log.Printf("[Controller] New machine successfully created.\n")

	} else if err != nil {
		log.Printf("[Controller] Error while checking for free machine for label %s. Error: %s\n", label, err)
		return "", err
	}

	log.Printf("[Controller] Trying to start new machine with the id %s in path %s.\n", id, workingPath)
	if err := c.VagrantConnector.StartMachineIn(workingPath); err != nil {
		log.Printf("[Controller] Error while spining up the box for label %s. Error: %s\n", label, err)
		return "", err
	}

	if err := c.Database.UpdateMachine(id, &DbMachine{id, label, label, "running", 1, "", "", ""}); err != nil {
		log.Printf("[Controller] Error while storing new machine status für id %s. Error: %s", id, err)
		return "", err
	}

	return id, nil
}

func (c *Controller) checkUnusedMachineFor(label string, baseDir string) (string, error) {
	machines, err := c.Database.GetMachines()
	if err != nil {
		log.Printf("[Controller] Error while retriving running machines status. Error: %s\n", err)
		return "", err
	}

	for _, m := range machines {
		if m.Label == label && m.State != "unused" {
			log.Printf("[Controller]\t=> Found machine %s for the label %s with the state %s!\n", m.ID, label, m.State)
			return filepath.Join(baseDir, m.ID), nil
		}
	}
	/*
		newID := generateFolderName(label)
		currTime := time.Now().Format(time.RFC3339)
		newMachine := DbMachine{newID, label, label, "not created", 1, currTime, currTime, ""}
		machinesArr := make([]DbMachine, 1)
		machinesArr[0] = newMachine

		if err := c.Database.InsertNewMachine(machinesArr); err != nil {
			log.Printf("[Controller] Error while saving machine with id %s in database. Error: %s\n", newID, err)
			return "", err
		}
	*/
	log.Printf("[Controller]\t=> Couldn't find a suitable machine for label %s.\n", label)
	return "", NoFreeMachineError
}

func generateFolderName(label string) string {
	currtime := time.Now().Format(time.RFC3339Nano)
	var buffer bytes.Buffer
	buffer.WriteString(label)
	buffer.WriteString(currtime)

	hash := sha1.New()
	hash.Write(buffer.Bytes())
	sum := hash.Sum(nil)

	return hex.EncodeToString(sum)
}

func checkFreeSysMem(boxMemory uint64) error {
	freeMemory := getFreeMemory()
	if boxMemory >= freeMemory {
		return &NoFreeMemoryError{freeMemory, boxMemory}
	}
	return nil
}

func (c *Controller) checkMaxVMCount() error {
	maxVMCount := c.Config.MaxVms
	vmCount, err := c.VagrantConnector.GetRunningMachineCount()
	if err != nil {
		log.Printf("[Controller] Error while getting number of running machines. Error: %s\n", err)
		return err
	}

	log.Printf("[Controller] %d boxes are running, allowed to run %d boxes", vmCount, maxVMCount)
	if vmCount+1 > maxVMCount {
		return &TooManyVmsError{vmCount + 1, maxVMCount}
	}
	return nil
}

func getFreeMemory() uint64 {
	mem := sigar.Mem{}
	mem.Get()

	log.Printf("[Controller] Currently free system memory %d\n", mem.ActualFree)
	return mem.ActualFree
}

// DestroyVms takes an id for a machine as parameter. It queries the database for the machine identifed by the passed id. If a machine was found the workingpath directory will be used to destroy the machine inside that directory.
func (c *Controller) DestroyVms(id string) error {
	m, err := c.Database.GetMachineWithID(id)
	if err != nil {
		log.Printf("[Controller] Error while loading machine with id %s form database. Error: %s\n", err)
		return err
	}

	if m.State != "running" {
		return fmt.Errorf("Machine with id %s not running.", id)
	}

	workingPath := filepath.Join(c.Config.WorkingDirPath, id)

	state, err := c.VagrantConnector.DestroyMachineFor(workingPath)
	if err != nil {
		log.Printf("[Controller] Error while destroying the machine in path %s. Error: %s\n", workingPath, err)
		return err
	}
	m.State = state
	if err := c.Database.UpdateMachine(id, m); err != nil {
		log.Printf("[Controller] Error while updating state of machine with id %s in database. Error: %s\n", id, err)
		return err
	}

	return nil
}
