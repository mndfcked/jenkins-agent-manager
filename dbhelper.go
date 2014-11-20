package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Slice of database migration patches. Every change on the database schema results in a new patch that's added to the slice.
var patches = [...]string{"CREATE TABLE machines (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, label TEXT NOT NULL, state TEXT NOT NULL, version INTEGER NOT NULL, createdAt TEXT NOT NULL, modifiedAt TEXT);",
	"ALTER TABLE machines ADD COLUMN snapshotid TEXT;"}

// DbHelper struct contains the DB file path, version number and a reference to the sql.DB struct
type DbHelper struct {
	Path    string
	Version uint
	Db      *sql.DB
}

// DbMachine is a struct that the result ob a database query encapsulates
type DbMachine struct {
	ID         string
	Name       string
	Label      string
	State      string
	Version    uint
	CreatedAt  string
	ModifiedAt string
	SnapshotID string
}

// NewDbHelper creates a new instance ob the DbHelper struct. It opens a new sql.Db instance and stores it in the DbHelper struct.
// If a new version is requested the database will be updated to this version
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

// GetMachines querys the database for alle machines stored in it. After a successfully executed query a slice of DbMachine structs will be returned.
func (h *DbHelper) GetMachines() ([]DbMachine, error) {
	const selectQuery = "SELECT id, name, label, state state, version, createdAt, modifiedAt, snapshotid FROM machines;"

	rows, err := h.Db.Query(selectQuery)
	if err != nil {
		log.Printf("[DbHelper] Error while querying the maschines database. Error: %s\n", err)
		return nil, err
	}
	defer rows.Close()

	var machines []DbMachine
	for rows.Next() {
		var id string
		var name string
		var label string
		var state string
		var version uint
		var createdAt string
		var modifiedAt string
		var snapshotID sql.NullString

		if err := rows.Scan(&id, &name, &label, &state, &version, &createdAt, &modifiedAt, &snapshotID); err != nil {
			log.Printf("[DbHelper] Error while retriving machines. Error: %s\n", err)
		}

		machines = appendMachine(machines, DbMachine{id, name, label, state, version, createdAt, modifiedAt, snapshotID.String})
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

// InsertNewMachine takes a slice with DbMachine structs as parameter and stores them in the underlying database.
func (h *DbHelper) InsertNewMachine(m []DbMachine) error {
	const insertQuery = "INSERT INTO machines (id, name, label, state, version, createdAt, modifiedAt, snapshotID) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"

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
		_, err := stmt.Exec(machine.ID, machine.Name, machine.Label, machine.State, machine.Version, currTime, currTime, machine.SnapshotID)
		if err != nil {
			log.Printf("[DbHelper] Error while executing insertion. Error: %s\n", err)
		}
	}

	tx.Commit()

	return nil
}

// UpdateMachine updates the database entry for the machine with the id that is passed to the function. The informations are taken from the reference to the DbMachine struct from the second parameter
func (h *DbHelper) UpdateMachine(id string, data *DbMachine) error {
	const updateQuery = "UPDATE OR FAIL machines SET id = ?, name = ?, label = ?, state = ?, version = ?, createdAt = ?, modifiedAt = ?, snapshotid = ? WHERE id = ?;"

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
	if _, err := stmt.Exec(data.ID, data.Name, data.Label, data.State, data.Version, data.CreatedAt, currTime, data.SnapshotID, id); err != nil {
		log.Printf("[DbHelper] Error while executing update statement for %s. Error: %s\nData: %#v\n", id, err, data)
		return err
	}

	tx.Commit()

	return nil
}

// DeleteMachine deletes the machine identified by the id that is passed as the paramater.
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

// GetMachineWithID querys the database for the machine identified by the passed id parameter and returns a reference to a DbMachine struct containing the information.
func (h *DbHelper) GetMachineWithID(id string) (*DbMachine, error) {
	const selectQuery = "SELECT id, name, label, state, version, createdAt, modifiedAt, snapshotid FROM machines WHERE id = ?"

	row := h.Db.QueryRow(selectQuery, id)

	if row == nil {
		log.Printf("[DbHelper] Error, machine with id %s not found.\n", id)
		return nil, fmt.Errorf("Machine with id %s not found.", id)
	}

	var machineID string
	var name string
	var label string
	var state string
	var version uint
	var createdAt string
	var modifiedAt string
	var snapshotID string

	if err := row.Scan(&machineID, &name, &label, &state, &version, &createdAt, &modifiedAt, &snapshotID); err != nil {
		log.Printf("[DbHelper] Error while querying for machine with id %s. Error: %s\nQuery: %s\n", id, err, selectQuery)
		return nil, err
	}

	return &DbMachine{machineID, name, label, state, version, createdAt, modifiedAt, snapshotID}, nil
}
