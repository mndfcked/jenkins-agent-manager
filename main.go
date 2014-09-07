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
	defaultServerUrl   = "http://localhost:8080"
	flagServerUrlUsage = "Url of the jenkins installation to coordinate"
	flagSecretUsage    = "Secret to use for auth to the server"
)

var serverUrl string
var serverSecret string

func init() {
	flag.StringVar(&serverUrl, "serverUrl", defaultServerUrl, flagServerUrlUsage)
	flag.StringVar(&serverSecret, "serverSecret", "e335641b4fde2caad93e7144ae046c82", flagSecretUsage)
}

func main() {
	flag.Parse()

	fmt.Println("====  Service started with the following config ====")
	fmt.Printf("ServerUrl: %v\n", serverUrl)
	fmt.Printf("ServerSecret: %v\n", serverSecret)

	fmt.Print("\n==== Trying to fetch jenkins information from ", serverUrl, " ...")
	jc := NewJenkinsConnector(serverUrl, serverSecret)
	comp, err := jc.requestComputerInfos()
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

	fmt.Println("\n==== Trying to create a http listener ====")
	contr, err := NewController()
	if err != nil {
		panic(err)
	}
	l, err := NewListener(":8888", contr)
	if err != nil {
		panic(err)
	}
	if l.CreateSocket(":8888"); err != nil {
		log.Fatal(err)
	}
}
