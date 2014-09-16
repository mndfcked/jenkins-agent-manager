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
	"fmt"
	"log"
	"net/http"
)

/*
 * Listener creates a socket to listen on a specified port and holds a reference to the controller to communicate to
 */
type Listener struct {
	Controller *Controller
}
/*
 * NewListener creates and returns a new Listener struct
 * TODO: Refactor to check for errors and return if necessary
 */
func NewListener(port string, controller *Controller) (*Listener, error) {
	return &Listener{controller}, nil
}

/*
 * CreateSocket creates a http socket for the listener on the specified port
 * TODO: Refactor and split up in better reusable parts
 */
func (l *Listener) CreateSocket(port string) error {
	// Inline definition of the handler func for the start command
	startHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vmLabel := r.FormValue("label")
		log.Printf("[LISTENER]: Trying to start a box for label %s.\n", vmLabel)
		if err := l.Controller.StartVms(vmLabel); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("[LISTENER]: Couldn't start the requested VM. ERROR: %s\n", err)
			return
		}
		fmt.Fprintf(w, "Successfully stated the box for label %s", vmLabel)
		log.Printf("[LISTENER]: Successfully started a box for label %s.\n", vmLabel)
	})
	
	// Inline definition of the handler func for the destroy command
	destroyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vmLabel := r.FormValue("label")
		log.Printf("[LISTENER]: Trying to the box for the label %s.\n", vmLabel)
		if err := l.Controller.DestroyVms(vmLabel); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("[LISTENER]: Couldn't destroy the requested VM. ERROR: %s\n", err)
			return
		}
		fmt.Fprintf(w, "Successfully destroyed the box for label %s", vmLabel)
		log.Printf("[LISTENER]: Successfully destroyed a box for label %s.\n", vmLabel)
	})

	http.Handle("/start", startHandler)
	http.Handle("/destroy", destroyHandler)

	if err := http.ListenAndServe(":8888", nil); err != nil {
		return err
	}

	return nil
}
