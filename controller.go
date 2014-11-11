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
	"log"

	"github.com/cloudfoundry/gosigar"
)

var (
	ErrTooManyVms = errors.New("Too many vms are running")
	ErrNoMemory   = errors.New("Not enough system memory available")
)

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
	maxVmCount := c.Config.MaxVms
	vmCount, err := c.VagrantConnector.GetRunningMachineCount()
	if err != nil {
		log.Printf("[Controller] Error while getting number of running machines. Error: %s\n", err)
		return err
	}

	log.Printf("[Controller] %d boxes are running, allowed to run %d boxes", vmCount, maxVmCount)
	if vmCount+1 > maxVmCount {
		log.Printf("[Controller] ERROR: Too many VMs are running")
		return ErrTooManyVms
	}
	freeMemory := getFreeMemory()
	boxMemory, err := c.VagrantConnector.GetBoxMemoryFor(label)
	if err != nil {
		log.Printf("[Controller] ERROR: Can't get required system memory for box with label %s.\n")
		return err
	}

	log.Printf("[Controller] System has %d Bytes free memory, box %s needs %d Bytes of system memory.\n", freeMemory, label, boxMemory)
	if uint64(boxMemory) >= freeMemory {
		log.Printf("[Controller] ERROR got only %d byte free mem, %d byte needed", freeMemory, boxMemory)
		return ErrNoMemory
	}

	if err := c.VagrantConnector.StartMachineFor(label, c.Config.WorkingDirPath); err != nil {
		log.Printf("[Contr]: ERROR: Error while spining up the box for label %s.\n", label)
		return err
	}

	return nil
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
