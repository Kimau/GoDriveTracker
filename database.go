package main

import (
	"encoding/json"
	"log"

	"github.com/boltdb/bolt"

	drive "google.golang.org/api/drive/v2"
)

var world = []byte("docList")
var boltDB *bolt.DB

func OpenDB(filename string) {
	var err error

	boltDB, err = bolt.Open(filename, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WriteFile(file *drive.File) {

	writeFileFunc := func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(world)
		if err != nil {
			return err
		}

		dat, e2 := json.Marshal(file)

		e2 = bucket.Put([]byte(file.Id), dat)
		if e2 != nil {
			return err
		}
		return nil
	}

	// store some data
	txErr := boltDB.Update(writeFileFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}
}

func LoadFile(fileId string) *drive.File {
	var result *drive.File = nil

	loadFileFunc := func(tx *bolt.Tx) error {
		bucket := tx.Bucket(world)
		if bucket == nil {
			log.Printf("Bucket %q not found!", world)
			return nil
		}

		dat := bucket.Get([]byte(fileId))
		if dat != nil {
			errMarshal := json.Unmarshal(dat, result)
			if errMarshal != nil {
				log.Println("Unmarshal failed:", errMarshal)
				return nil
			}

		}

		return nil
	}

	// retrieve the data
	txErr := boltDB.View(loadFileFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}

	return result
}
