package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"testing"
)

func TestDb(t *testing.T) {
	log.Println("=> Testing dbhelper")

	const path = "./test.db"
	const dbVersion = 1
	log.Printf("\t=> Cleaning test data db at %s\n", path)
	if err := os.Remove(path); err != nil {
		t.Errorf("\t=> Error while cleaning test data at %s. Error: %s\n", path, err)
	}

	log.Printf("\t=> Creating new DbHelper instance with version %d at path %s\n", dbVersion, path)
	helper, err := NewDbHelper(path, dbVersion)
	if err != nil {
		t.Errorf("\t=> Fail while getting DbHelper. Error: %s", err)
	}

	log.Println("\t=> Requesting empty array of machines from test db")
	machines, err := helper.GetRunningMachines()
	if err != nil {
		t.Errorf("\t=> Fail while getting empty running machines array. Error: %s", err)
	}

	if len(machines) != 0 {
		t.Errorf("\t=> Expected empty reply, got %d items", len(machines))
	}

	log.Println("\t=> Inserting test data into test db.")
	mArr := make([]DbMachine, 10)
	for i := 0; i < 10; i++ {
		str := fmt.Sprintf("Lorem impsum %d", i)
		hash := sha256.New()
		hash.Write([]byte(str))
		sum := hash.Sum(nil)
		sumStr := hex.EncodeToString(sum)
		mArr[i] = DbMachine{Id: sumStr, Name: "foo", Label: "baz", State: "bar", Version: uint(1), CreatedAt: "bla", ModifiedAt: "bla"}
	}

	helper.InsertNewMachine(mArr)

	insertedMachines, err := helper.GetRunningMachines()
	if err != nil {
		t.Error("\t=> Fail while getting running machines test data from db")
	}
	for key, m := range insertedMachines {
		log.Printf("key: %d, machine: %#v\n", key, m)
	}

	log.Printf("\t=> Changing the test data\n")
	for key, m := range insertedMachines {
		m.Name = "bi"
		m.Label = "ba"
		m.State = "butzeman"
		m.Version = 10815
		m.ModifiedAt = "fooobarrr"
		if err := helper.UpdateMachine(m.Id, &m); err != nil {
			t.Errorf("Fail while updating entry %d. Error: %s\n", key, err)
		}
	}

	updatedMachines, err := helper.GetRunningMachines()
	if err != nil {
		t.Error("\t=> Fail while getting running machines test data from db")
	}
	for key, m := range updatedMachines {
		log.Printf("key: %d, machine: %#v\n", key, m)
	}

	log.Printf("\t=> Deleting test data\n")
	for key, m := range updatedMachines {
		if err := helper.DeleteMachine(m.Id); err != nil {
			t.Errorf("Fail while deleting entry whith id %d. Error: %s\n", key, err)
		}
	}

	deletedMachines, err := helper.GetRunningMachines()
	if err != nil {
		t.Error("\t=> Fail while getting running machines test data from db")
	}
	for key, m := range deletedMachines {
		log.Printf("key: %d, machine: %#v\n", key, m)
	}
}
