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

//TODO: Implement new design
// 	- save started VMs in a file or sqlitedb
//	- start every vm in a new folder with a sha256 generated name (=> vagrantconnector)
//	- check before every start of a new machine, wheter it there is a free, already created machine available
//	- return the sha256 folder name as id (=> listener)
//	- requere an id for destroying a machine (=>listener)
//	- don't destroy a machine, snapshot and restore instead (=> vagrantconnector, only when already created)

import (
	"fmt"
	"log"

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

// Controller struct gives other type to hold reference to it
type Controller struct {
	VagrantConnector *VagrantConnector
	JenkinsConnector *JenkinsConnector
	Config           *Configuration
}

// NewController instatiates a new Controller and returns it
func NewController(vc *VagrantConnector, jc *JenkinsConnector, conf *Configuration) (*Controller, error) {
	return &Controller{vc, jc, conf}, nil
}

func (c *Controller) StartVms(label string) error {
	log.Printf("[Controller] Received request to start a box for label %s.\n", label)

	boxMemory, err := c.VagrantConnector.GetBoxMemoryFor(label)
	if err != nil {
		return err
	}

	if err := c.checkMaxVMCount(); err != nil {
		log.Printf("[Controller] Error while checking for max. concurrently allowed machines. Error: %s\n", err)
		return err
	}

	if err := checkFreeSysMem(uint64(boxMemory)); err != nil {
		log.Printf("[Controller] Error while checking free system memory for box %s. Error: %s\n", label, err)
		return err
	}

	if err := c.VagrantConnector.StartMachineFor(label, c.Config.WorkingDirPath); err != nil {
		log.Printf("[Controller] Error while spining up the box for label %s. Error: %s\n", label, err)
		return err
	}

	return nil
}

func checkFreeSysMem(boxMemory uint64) error {
	freeMemory := getFreeMemory()
	if boxMemory >= freeMemory {
		return &NoFreeMemoryError{freeMemory, boxMemory}
	} else {
		return nil
	}
}

func (c *Controller) checkMaxVMCount() error {
	maxVmCount := c.Config.MaxVms
	vmCount, err := c.VagrantConnector.GetRunningMachineCount()
	if err != nil {
		log.Printf("[Controller] Error while getting number of running machines. Error: %s\n", err)
		return err
	}

	log.Printf("[Controller] %d boxes are running, allowed to run %d boxes", vmCount, maxVmCount)
	if vmCount+1 > maxVmCount {
		return &TooManyVmsError{vmCount + 1, maxVmCount}
	} else {
		return nil
	}
}

func getFreeMemory() uint64 {
	mem := sigar.Mem{}
	mem.Get()

	log.Printf("[Controller] Currently free system memory %d\n", mem.ActualFree)
	return mem.ActualFree
}

func (c *Controller) DestroyVms(label string) error {
	if err := c.VagrantConnector.DestroyMachineFor(label, c.Config.WorkingDirPath); err != nil {
		log.Printf("[Controller]: Error while destroying the boxes for %s\n", label)
		return err
	}
	return nil
}
