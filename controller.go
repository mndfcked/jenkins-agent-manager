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

import "errors"

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

func (c *Controller) StartVms(count int) (int, error) {
	maxVmCount := c.Config.MaxVms
	vmCount := c.VagrantConnector.GetVmCount()
	if maxVmCount >= vmCount {
		return 0, ErrTooManyVms
	}

	freeMemory, err := c.JenkinsConnector.GetFreeSystemMemory()
	if err != nil {
		return 0, err
	}
	boxMemory := c.VagrantConnector.GetBoxMemory()

	if boxMemory*int64(count) <= freeMemory {
		return c.VagrantConnector.SpinUpNew(vmCount)
	}
	return 0, ErrNoMemory

}
