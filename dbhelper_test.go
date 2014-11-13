package main

import (
	"fmt"
	"os"
	"testing"
)

func TestDb(t *testing.T) {

	fmt.Printf("Test")
	const path = "./test.db"
	os.Remove(path)
	helper, err := NewDbHelper(path, 2)
	if err != nil {
		t.Errorf("Fail while getting DbHelper. Error: %s", err)
	}

	db := helper.Database

	if err := helper.GetRunningMachines(); err != nil {
		t.Error(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Error(err)
	}

	stmt, err := tx.Prepare("insert into machines(id, label) values(?, ?)")
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 100; i++ {
		_, err := stmt.Exec(i, fmt.Sprintf("Lipsum nr %d", i))
		if err != nil {
			t.Error(err)
		}

	}

	if err := tx.Commit(); err != nil {
		t.Error(err)
	}

	if err := helper.GetRunningMachines(); err != nil {
		t.Error(err)
	}

}
