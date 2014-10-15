jenkins-agent-manager (jam)
======================

[![Build Status](https://travis-ci.org/mndfcked/jenkins-agent-manager.svg?branch=master)](https://travis-ci.org/mndfcked/jenkins-agent-manager)

A simple service which manages Vagrant boxes based on the load of the Jenkins master and requested box.

# Setup
* `git clone https://github.com/mndfcked/jenkins-agent-manager.git` into your `$GOPATH/src` directory.
* `cd` into the cloned folder, run `go install` and `jenkins-agent-manager` or `go run *.go` 

# Configuration
jam expects a JSON formatted configuration file. The default path is set to `/etc/jenkins-agent-manager/config.json`, but you can use the flag `-confPath="/path/to/your/config/file.json"` to pass a custom path to jam.
The sample configuration below lists all currently available options.
```JSON
{
  "jenkins_api_url":"http://localhost:8080",
  "jenkins_api_secret":"",
  "listener_port":"8888",
  "max_vm_count":2,
  "working_dir_path":"/tmp",
  "boxes":[
    {
      "name": "win7-slave",
      "labels": ["windows", "windows7"],
      "memory": "2048MB"    
    },
    {
     "name": "centos7-slave",
     "labels": ["linux", "centos7", "centos"],
     "memory": "2048MB"
    }  
  ]
}
```

* `jenkins_api_url`
  * The Url of the Jenkins API.
* `jenkins_api_secret`
  * The secret to authenticate with the Jenkins API.
* `listener_port` 
  * The port jam is listening on for requests.
* `mac_vm_count`
  * The number of vagrant boxes that can be run at the same time.
* `working_dir_path`
  * The path where jam creates the vagrant enviroments for the started boxes.
* `boxes`
  * A JSON-Array with JSON-Objects describing a vagrant box jam can use. `name` is the name of the box as provided to the `vagrant box add "name" "box"` command. labels is a JSON-Array of string that are used to identify the box to start.
  * `name`: The name of the box.
  * `labels`: The labels identifing the capabillities of the box.
  * `memory`: The amount of system memory the box will be using.

# Note
This is part of my bachelor thesis and still work in progress.

# Contributions
Currently not desirable.
