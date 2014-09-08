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
func NewController(vc *VagrantConnector, jc *JenkinsConnector, conf *Configuration) *Controller {
	return &Controller{vc, jc, conf}
}

func (c *Controller) StartVms(count uint64) (int, error) {
	log.Printf("[VC]: Received request to start %d boxes.\n", count)
	maxVmCount := c.Config.MaxVms
	vmCount := c.VagrantConnector.GetVmCount()

	log.Printf("[VC]: %d boxes are running, allowed to run %d boxes", vmCount, maxVmCount)
	if vmCount+int(count) > maxVmCount {
		log.Printf("[VC]: ERROR: Too many VMs are running")
		return 0, ErrTooManyVms
	}

	freeMemory, err := c.JenkinsConnector.GetFreeSystemMemory()
	if err != nil {
		log.Printf("[VC]: ERROR: Can't get the free system memory")
		return 0, err
	}
	boxMemory := c.VagrantConnector.GetBoxMemory()

	neededMem := boxMemory * int64(count)
	if neededMem >= freeMemory {
		log.Printf("[VC]: ERROR got only %d byte free mem, %d byte needed", freeMemory, neededMem)
		return 0, ErrNoMemory
	}

	startedCount, err := c.VagrantConnector.SpinUpNew(vmCount, c.Config.BoxPath)
	if err != nil {
		log.Printf("[VC]: ERROR: Error while spining up the boxes. %d boxes started", startedCount)
		return startedCount, err
	}

	return startedCount, nil
}
