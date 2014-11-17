package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var patches = [...]string{"CREATE TABLE machines (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, label TEXT NOT NULL, state TEXT NOT NULL, version INTEGER NOT NULL, createdAt TEXT NOT NULL, modifiedAt TEXT);"}

type DbHelper struct {
	Path    string
	Version uint
	Db      *sql.DB
}

type DbMachine struct {
	Id         string
	Name       string
	Label      string
	State      string
	Version    uint
	CreatedAt  string
	ModifiedAt string
}

func NewDbHelper(path string, version uint) (*DbHelper, error) {

	if version > uint(len(patches)) {
		return nil, fmt.Errorf("The requested version %d is to high", version)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Printf("[DbHelper] Error while opening database %s. Error: %s\n", path, err)
		return nil, err
	}

	oldVersion, err := getVersion(db)
	if oldVersion < version {
		if err := updateSchema(db, oldVersion, version); err != nil {
			log.Println(err)
			return nil, err
		}

	}

	return &DbHelper{path, version, db}, nil
}

func setVersion(db *sql.DB, version uint) {
	var userVersionPragma = fmt.Sprintf("PRAGMA user_version = %d;", version)
	if _, err := db.Exec(userVersionPragma); err != nil {
		log.Printf("[DbHelper] Error while setting database version. Error:%s\nQuery:%s\n", err, userVersionPragma)
	}
}

func getVersion(db *sql.DB) (uint, error) {
	const getUserVersionQuery = "PRAGMA user_version;"
	row := db.QueryRow(getUserVersionQuery)

	if row == nil {
		log.Fatalf("%s not found", getUserVersionQuery)
		return 0, fmt.Errorf("Result row for Query: %s was empty.", getUserVersionQuery)
	}

	var version int
	if err := row.Scan(&version); err != nil {
		log.Printf("[DbHelper] Error while getting database version pragma. Error: %s\nQuery: %s\n", err, getUserVersionQuery)
		return 0, err
	}

	return uint(version), nil
}

func updateSchema(db *sql.DB, oldVersion uint, newVersion uint) error {
	log.Printf("[DbHelper] Trying to update database schema from old version %d to the new version %d", oldVersion, newVersion)

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("[DbHelper] Error while starting transaction for db. Error: %s\n", err)
	}

	for key, patch := range patches {
		log.Printf("[DbHelper] Schema Update => Patch %d: %s\n", key, patch)
		if uint(key+1) > oldVersion {
			_, err := tx.Exec(patch)
			if err != nil {
				log.Printf("[DbHelper] Error while applying patch %s. Error: %s\n", patch, err)
				if err := tx.Rollback(); err != nil {
					log.Printf("[DbHelper] Error while rolling back transaction. Error: %s\n", err)
				}
				return err
			}
		}
	}

	tx.Commit()
	setVersion(db, newVersion)
	return nil
}

func (h *DbHelper) GetRunningMachines() ([]DbMachine, error) {
	const selectQuery = "SELECT id, name, label, state state, version, createdAt, modifiedAt FROM machines;"

	rows, err := h.Db.Query(selectQuery)
	if err != nil {
		log.Printf("[DbHelper] Error while querying the maschines database. Error: %s\n", err)
		return nil, err
	}
	defer rows.Close()

	machines := make([]DbMachine, 0)
	for rows.Next() {
		var id string
		var name string
		var label string
		var state string
		var version uint
		var createdAt string
		var modifiedAt string

		if err := rows.Scan(&id, &name, &label, &state, &version, &createdAt, &modifiedAt); err != nil {
			log.Printf("[DbHelper] Error while retriving label. Error: %s\n", err)
		}

		machines = appendMachine(machines, DbMachine{id, name, label, state, version, createdAt, modifiedAt})
	}

	return machines, nil
}

func appendMachine(machines []DbMachine, data ...DbMachine) []DbMachine {
	currLen := len(machines)
	newLen := currLen + len(data)

	if newLen > cap(machines) {
		newMachines := make([]DbMachine, (newLen+1)*2)
		copy(newMachines, machines)
		machines = newMachines
	}
	machines = machines[0:newLen]
	copy(machines[currLen:newLen], data)

	return machines
}

func (h *DbHelper) InsertNewMachine(m []DbMachine) error {
	const insertQuery = "INSERT INTO machines (id, name, label, state, version, createdAt, modifiedAt) VALUES (?, ?, ?, ?, ?, ?, ?)"

	tx, err := h.Db.Begin()
	if err != nil {
		log.Printf("[DbHelper] Error while starting transaction for inserting a new machine. Error: %s\n", err)
		return err
	}

	stmt, err := tx.Prepare(insertQuery)
	if err != nil {
		log.Printf("[DbHelper] Error while preparing insert statement %s. Error: %s\n", insertQuery, err)
	}
	defer stmt.Close()

	currTime := time.Now().Format(time.RFC3339)
	for _, machine := range m {
		_, err := stmt.Exec(machine.Id, machine.Name, machine.Label, machine.State, machine.Version, currTime, currTime)
		if err != nil {
			log.Printf("[DbHelper] Error while executing insertion. Error: %s\n", err)
		}
	}

	tx.Commit()

	return nil
}

func (h *DbHelper) UpdateMachine(id string, data *DbMachine) error {
	const updateQuery = "UPDATE OR FAIL machines SET id = ?, name = ?, label = ?, state = ?, version = ?, createdAt = ?, modifiedAt = ? WHERE id = ?;"

	tx, err := h.Db.Begin()
	if err != nil {
		log.Printf("[DbHelper] Error while starting transaction for updating %s. Error: %s\n", id, err)
		return err
	}

	stmt, err := tx.Prepare(updateQuery)
	if err != nil {
		log.Printf("[DbHelper] Error while preparing update statement for. Error: %s\n", id, err)
		return err
	}
	defer stmt.Close()

	currTime := time.Now().Format(time.RFC3339)
	if _, err := stmt.Exec(data.Id, data.Name, data.Label, data.State, data.Version, data.CreatedAt, currTime, id); err != nil {
		log.Printf("[DbHelper] Error while executing update statement for %s. Error: %s\nData: %#v\n", id, err, data)
		return err
	}

	tx.Commit()

	return nil
}

func (h *DbHelper) DeleteMachine(id string) error {
	const deleteQuery = "DELETE FROM machines WHERE id = ?;"

	tx, err := h.Db.Begin()
	if err != nil {
		log.Printf("[DbHelper] Error while starting transaction for deleting %s. Error: %s\n", id, err)
		return err
	}

	stmt, err := tx.Prepare(deleteQuery)
	if err != nil {
		log.Printf("[DbHelper] Error while preparing deletion for %s statement. Error: %s\n", id, err)
		return err
	}
	defer stmt.Close()

	if _, err := stmt.Exec(id); err != nil {
		log.Printf("[DbHelper] Error while executing deletion statement for %s. Error: %s\n", id, err)
		return err
	}

	tx.Commit()

	return nil
}
