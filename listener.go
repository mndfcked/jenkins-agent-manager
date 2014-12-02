//
// Copyright [2014] [Jörn Domnik]
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http//www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// Listener creates a socket to listen on a specified port and holds a reference to the controller to communicate to
type Listener struct {
	Port       string
	Controller *Controller
}

// Response is a struct that encapsulates all informations that a posible listener response can contain.
type Response struct {
	State string
	ID    string
	Label string
}

// NewListener creates and returns a new Listener struct
func NewListener(port string, controller *Controller) *Listener {
	//TODO: Refactor to check for errors and return if necessary
	return &Listener{port, controller}
}

// CreateSocket creates a http socket for the listener on the specified port
func (l *Listener) CreateSocket() error {
	//TODO: Refactor and split up in better reusable parts
	log.Printf("[Listener] Creating new listening socket for the address %s.\n", l.Port)
	// Inline definition of the handler func for the start command
	startHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vmLabel := r.FormValue("label")
		log.Printf("[Listener]: Trying to start a box for label %s.\n", vmLabel)
		id, err := l.Controller.StartVms(vmLabel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("[Listener]: Error whiel starting the requested machine. Error: %s\n", err)
			return
		}

		response, err := createResponse("success", id, vmLabel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("[Listener] Error while creating response JSON. Error: %s\n", err)
			return
		}

		w.Header().Set("Server", "Jenkins Agent Manager")
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)

		log.Printf("[Listener]: Successfully started machine with id %s for label %s. Response: %s\n", id, vmLabel, response)
	})

	// Inline definition of the handler func for the destroy command
	destroyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue("id")
		log.Printf("[Listener]: Got request to destroy the machine with the id %s.\n", id)
		if err := l.Controller.DestroyVms(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("[Listener] Error while destroying the machine with id %s. Error: %s\n", id, err)
			return
		}

		response, err := createResponse("success", id, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("[Listener] Error while creating response JSON. Error: %s\n")
			return
		}

		w.Header().Set("Server", "Jenkins Agent Manager")
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)

		log.Printf("[Listener]: Successfully destroyed the machine with the id %s.\n", id)
	})

	http.Handle("/start", startHandler)
	http.Handle("/destroy", destroyHandler)

	if err := http.ListenAndServe(l.Port, nil); err != nil {
		return err
	}

	log.Println("[Listener] Listener successfully created. Listening on handlers /start and /destroy")

	return nil
}

func createResponse(state string, id string, label string) ([]byte, error) {
	response := Response{state, id, label}
	return json.Marshal(response)
}
