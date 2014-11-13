package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var patches = [...]string{"CREATE TABLE machines (id text not null primary key, label text, createdAt integer);",
	"CREATE TABLE zweite (id text not null primary key, label text, createdAt integer);",
	"CREATE TABLE dreote (id text not null primary key, label text, createdAt integer);"}

type DbHelper struct {
	Database *sql.DB
}

func NewDbHelper(path string, version uint) (*DbHelper, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	oldVersion, err := getVersion(db)
	if oldVersion < version {
		if err := updateSchema(db, oldVersion, version); err != nil {
			log.Println(err)
			return nil, err
		}

	}

	return &DbHelper{db}, nil
}

func setVersion(db *sql.DB, version uint) {
	var userVersionPragma = fmt.Sprintf("PRAGMA user_version = %d;", version)
	if _, err := db.Exec(userVersionPragma); err != nil {
		log.Printf("[DbHelper] Error while setting database version. Error:%s\nQuery:%s\n", err, userVersionPragma)
	}
}

func getVersion(db *sql.DB) (uint, error) {
	var getUserVersionQuery = "PRAGMA user_version;"
	row, _ := db.Query(getUserVersionQuery)

	for row.Next() {
		version := make([]byte, 16)
		if err := row.Scan(&version); err != nil {
			log.Printf("[DbHelper] Error while getting database version pragma. Error: %s\nQuery: %s\n", err, getUserVersionQuery)
		}
		log.Printf("String: %s, Zahl: %d. Struct: %+v, %#v, HEx: %x", version, version, version, version, version)
	}

	return 1, nil
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

func getSchemaVersion(db *sql.DB) uint {
	log.Println("[DbHelper] Trying to get the schema version of the db")
	const schemaVersionQuery = "select version from schema"

	var version uint
	if err := db.QueryRow(schemaVersionQuery).Scan(&version); err != nil {
		log.Fatalf("[DbHelper] Error while query schema version table. Error: %s\nQuery: %s\n", err, schemaVersionQuery)
	}

	return version
}

func (h *DbHelper) GetRunningMachines() error {
	const maschinesQuery = "select label from machines"

	rows, err := h.Database.Query(maschinesQuery)
	if err != nil {
		log.Printf("[DbHelper] Error while querying the maschines database. Error: %s\n", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			log.Printf("[DbHelper] Error while retriving label. Error: %s\n", err)
		}

		log.Printf("[DbHelper] Found a label: %s\n", label)
	}
	return nil
}
