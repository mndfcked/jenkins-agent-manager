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
	"flag"
	"fmt"
	"log"
)

// config flags
const (
	defaultServerUrl    = "http://localhost:8080"
	usageServerUrl      = "Url of the jenkins installation to coordinate"
	defaultSecret       = ""
	usageSecret         = "Secret to use for auth to the server"
	defaultListenerPort = ":8888"
	usageListenerPort   = "Port configured in the jenkins-vm-coordinator plugin"
	defaultMaxVms       = 1
	usageMaxVms         = "The maximal number of vagrant machines that can be spun up"
)

var (
	serverUrl    string
	serverSecret string
	listenerPort string
	maxVms       int
)

func init() {
	flag.StringVar(&serverUrl, "serverUrl", defaultServerUrl, usageServerUrl)
	flag.StringVar(&serverSecret, "serverSecret", defaultSecret, usageSecret)
	flag.StringVar(&listenerPort, "listenerPort", defaultListenerPort, usageListenerPort)
	flag.IntVar(&maxVms, "maxVms", defaultMaxVms, usageMaxVms)
}

func main() {
	flag.Parse()

	fmt.Println("====  Service started with the following config ====")
	fmt.Printf("ServerUrl: %v\n", serverUrl)
	fmt.Printf("ServerSecret: %v\n", serverSecret)

	fmt.Println("\n==== Creating new configuration =====")
	conf := NewConfiguration(serverUrl, serverSecret, listenerPort, maxVms)

	fmt.Print("\n==== Trying to fetch jenkins information from ", serverUrl, " ...")
	jc := NewJenkinsConnector(serverUrl, serverSecret)
	comp, err := jc.requestComputerInfo()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n ... Successfully established connection and got the following information ====")
	comp.PrettyPrint()

	fmt.Println("\n==== Printing the datastructures ====")
	fmt.Printf("%+v", comp)

	fmt.Println("\n==== Trying to load vagrant enviroment information ====")
	vc, err := NewVagrantConnector()
	if err != nil {
		log.Fatal(err)
	}
	vc.Print()

	fmt.Println("\n==== Printing data structure ====")
	fmt.Printf("%+v", vc)

	fmt.Println("\n==== Trying to create HTTP listener and start controller ====")
	if startController(vc, jc, conf); err != nil {
		panic(err)
	}
}

func startController(vc *VagrantConnector, jc *JenkinsConnector, conf *Configuration) error {
	contr := NewController(vc, jc, conf)

	l, err := NewListener(conf.ListenerPort, contr)
	if err != nil {
		return err
	}
	return l.CreateSocket(conf.ListenerPort)
}
