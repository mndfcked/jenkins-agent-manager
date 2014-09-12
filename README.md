jenkins-agent-manager (jam)
======================

A simple go program whose main purpose is to manage vagrant boxes based on the load of the JenkinsCI master.

# Configuration
Currently configuration options are passed as aguments to the jam executable. Available options are listet below:
* ```jenkinsApiUrl``` (Default: localhost:8080)
  * Url of the Jenkins API to get the management information from.
* ```jenkinsApiSecret``` (Dafault: "") 
  * API secret for the Jenkins API.
* ```listenerPort``` (Defailt: ":8888")
  * Port the manager listener will listen for messages.
* ```maxVms``` (Default: "2") 
  * Number of boxes that can be run simultaneously.
* ```workingDir``` (Default: "/tmp") 
  * Path to the directory the manager uses to store the vagrant management files.

# Note
This is part of my bachelor thesis and still work in progress.
