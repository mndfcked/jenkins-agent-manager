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
	"net/http"
)

type hudsonSwapSpaceMonitor struct {
	AvailablePhysicalMemory int64 `json:"availablePhysicalMemory"`
	AvailableSwapSpace      int64 `json:"availableSwapSpace"`
	TotalPhysicalMemory     int64 `json:"totalPhysicalMemory"`
	TotalSwapSpace          int64 `json:"totalSwapSpace"`
}

type computerMonitorData struct {
	SwapSpaceMonitor hudsonSwapSpaceMonitor `json:"hudson.node_monitors.SwapSpaceMonitor"`
}

type executor struct {
	CurrentExecutable string `json:"currentExecutable"`
	CurrentWorkUnit   string `json:"currentWorkUnit"`
	Idle              bool   `json:"idle"`
	LikelyStuck       bool   `json:"likelyStuck"`
	Number            int    `json:"number"`
	Progress          int    `json:"progress"`
}

type computer struct {
	DisplayName string              `json:"displayName"`
	MonitorData computerMonitorData `json:"monitorData"`
	Executors   []executor          `json:"executors"`
}

type ComputerInfo struct {
	BusyExecutors  int        `json:"busyExecutors"`
	TotalExecutors int        `json:"totalExecutors"`
	Computers      []computer `json:"computer"`
}

type JenkinsConnector struct {
	BaseUrl   string
	AuthToken string
}

func NewJenkinsConnector(baseUrl string, authToken string) *JenkinsConnector {
	return &JenkinsConnector{baseUrl, authToken}
}

func (jc *JenkinsConnector) requestComputerInfo() (*ComputerInfo, error) {
	url := buildUrl(jc.BaseUrl, jc.AuthToken, "/computer/api/json?depth=2")

	resp, err := http.Get(url)
	fmt.Println(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var j ComputerInfo
	if err := json.Unmarshal(body, &j); err != nil {
		return nil, err
	}

	return &j, err
}

func buildUrl(url string, token string, path string) string {
	return url + path + "&token=" + token
}

func (computerInfo *ComputerInfo) PrettyPrint() {
	for _, c := range computerInfo.Computers {
		fmt.Println("====== Jenkins Infos ======")
		fmt.Printf("BusyExecutors: %d\nTotalExecutors: %d\n", computerInfo.BusyExecutors, computerInfo.TotalExecutors)
		fmt.Printf("\n===== DisplayName: %s ======\n", c.DisplayName)
		swm := c.MonitorData.SwapSpaceMonitor
		fmt.Printf("TotalPhysicalMemory %d\n", swm.TotalPhysicalMemory)
		fmt.Printf("AvailablePhysicalMemory: %d\n", swm.AvailablePhysicalMemory)
		fmt.Printf("AvailableSwapSpace: %d\n", swm.AvailableSwapSpace)
		fmt.Printf("TotalSwapSpace %d\n", swm.TotalSwapSpace)
	}
}

func (jc *JenkinsConnector) GetFreeSystemMemory() (int64, error) {
	c, err := jc.requestComputerInfo()
	if err != nil {
		return 0, err
	}
	for _, v := range c.Computers {
		if v.DisplayName == "master" {
			return v.MonitorData.SwapSpaceMonitor.AvailablePhysicalMemory, nil
		}
	}
	err = errors.New("Internal error")
	return 0, err
}
