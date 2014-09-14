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

/*
 * Definition of configuration constants
 */
const (
	defaultConfPath = "/etc/jenkins-agent-manager/config.json"
	usageConfPath   = "Path to the configuration file. Has to be valid JSON format"
)

/*
 * Definition of configuration variables
 */
var confPath string

/*
 * Initialize main program state
 */
func init() {
	/*
	 * TODO: Verify that vagrant is installed
	 * TODO: Verify that jenkins is installed and running
	 */
	flag.StringVar(&confPath, "configurationPath", defaultConfPath, usageConfPath)
}

/*
 * Main Program starting point
 */
func main() {
	/*
	 *
	 * TODO: Cache all machines from vagrant global-status
	 * TODO: Create routine that searches for the desired box type, if not existing -> create
	 * TODO: vagrant up on free boxes, cache internal which boxes are already used
	 * TODO: reset boxes to a snapshot after it was used
	 *
	 */
	flag.Parse()

	fmt.Println("==== Creating new configuration =====")
	conf, err := NewConfiguration(confPath)
	if err != nil {
		log.Panicf("[MAIN]: ERROR: Couldn't creat configuration.\nError: %s\n", err.Error())
	}
	log.Println("Configuration successfully created.")
	fmt.Println("====  Service started with the following config ====")
	log.Printf("Jenkins API-Url\t=>\t%+v\n", conf.JenkinsApiUrl)
	log.Printf("Jenkins API-Secret\t=>\t%+v\n", conf.JenkinsApiSecret)
	log.Printf("Listener port\t=>\t%+v\n", conf.ListenerPort)
	log.Printf("Max. VM count\t=>\t%+v\n", conf.MaxVms)
	log.Printf("Working directory\t=>\t%+v\n", conf.WorkingDirPath)
	log.Printf("Boxes\t=>\t%+v\n", conf.Boxes)
	fmt.Println("====================================================\n")

	fmt.Printf("==== Trying to fetch jenkins information from %s ====\n", conf.JenkinsApiUrl)
	jc, err := NewJenkinsConnector(conf.JenkinsApiUrl, conf.JenkinsApiSecret)
	if err != nil {
		log.Panicf("[MAIN]: ERROR: Couldn't create a JenkinsConnector instance.\nError: %s\n", err.Error())
	}
	log.Println("Successfully established connection and collected information.")
	fmt.Println("====================================================\n")

	fmt.Println("==== Trying to load vagrant enviroment information ====")
	vc, err := NewVagrantConnector(conf)
	if err != nil {
		log.Panicf("[MAIN]: ERROR: Couldn't create VagrantConnector instance.\nError: %s\n", err.Error())
	}
	log.Println("Successfully loaded vagrant enviroment.")
	fmt.Println("=======================================================\n")

	fmt.Println("==== Creating controller instance ====")
	contr, err := NewController(vc, jc, conf)
	if err != nil {
		log.Panicf("[MAIN]: ERROR: Couldn't create Controller instance.\nError: %s\n", err.Error())
	}
	log.Println("Successfully create controller instance.")
	fmt.Println("======================================\n")

	if createListener(conf, contr); err != nil {
		log.Panicf("[MAIN]: ERROR: Couldn't create HTTP listener.\nError: %s\n", err.Error())
	}
}

func createListener(conf *Configuration, c *Controller) error {
	l, err := NewListener(conf.ListenerPort, c)
	if err != nil {
		return err
	}
	return l.CreateSocket(conf.ListenerPort)
}
