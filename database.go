package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/boltdb/bolt"

	drive "google.golang.org/api/drive/v2"
)

var bucketDoc = []byte("doc")
var bucketRevs = []byte("revs")
var boltDB *bolt.DB

func OpenDB(filename string) {
	var err error

	boltDB, err = bolt.Open(filename, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	writeFileFunc := func(tx *bolt.Tx) error {
		{
			_, err := tx.CreateBucketIfNotExists(bucketDoc)
			if err != nil {
				return err
			}
		}
		{
			_, err := tx.CreateBucketIfNotExists(bucketRevs)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// store some data
	txErr := boltDB.Update(writeFileFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}

}

func CloseDB() {
	err := boltDB.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func WriteFile(file *drive.File) {
	writeFileFunc := func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketDoc)
		if err != nil {
			log.Println("Bucket failed:", err)
			return err
		}

		dat, eMarshal := json.Marshal(file)
		if eMarshal != nil {
			log.Println("Marhsal failed:", eMarshal)
			return eMarshal
		}

		ePut := bucket.Put([]byte(file.Id), dat)
		if ePut != nil {
			log.Println("Put failed:", ePut)
			return ePut
		}

		// log.Println("Write File:", file.Id, bucket.Stats())

		return nil
	}

	// store some data
	txErr := boltDB.Update(writeFileFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}
}

func WriteRevision(fileId string, rev *drive.Revision) {

	writeRevFunc := func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketRevs)
		if err != nil {
			return err
		}

		dat, e2 := json.Marshal(rev)

		e2 = bucket.Put([]byte(fileId+rev.Id), dat)
		if e2 != nil {
			return err
		}

		if *debug {
			log.Println("Write Revision:", rev.Id)
		}

		return nil
	}

	// store some data
	txErr := boltDB.Update(writeRevFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}
}

func LoadFile(fileId string) *drive.File {
	var result drive.File

	loadFileFunc := func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketDoc)
		if bucket == nil {
			log.Printf("Bucket %q not found!", bucketDoc)
			return errors.New("Bucket not found!")
		}

		dat := bucket.Get([]byte(fileId))
		if dat == nil {
			return errors.New("File not found")
		}

		errMarshal := json.Unmarshal(dat, &result)
		if errMarshal != nil {
			log.Println("Unmarshal failed:", errMarshal)
			return errMarshal
		}

		return nil
	}

	// retrieve the data
	txErr := boltDB.View(loadFileFunc)
	if txErr != nil {
		return nil
	}

	return &result
}

func DumpDocListKeys() {
	fmt.Println("Dump Doc List")

	dumpDBDocs := func(tx *bolt.Tx) error {

		dumpDoc := func(k, v []byte) error {
			var result drive.File
			errMarshal := json.Unmarshal(v, &result)

			if errMarshal != nil {
				fmt.Printf("%s is not parsed. \n %s \n %s \n", k, errMarshal.Error(), v)
				return errMarshal
			} else {
				fmt.Printf("%s \n\t Title: %s \n\t Last Mod: %s \n", k, result.Title, result.ModifiedDate)
			}

			return nil
		}

		bucket := tx.Bucket(bucketDoc)
		log.Println(bucket.Stats())

		err := bucket.ForEach(dumpDoc)
		if err != nil {
			log.Println(err)
			return err
		}

		return nil
	}

	// retrieve the data
	txErr := boltDB.View(dumpDBDocs)
	if txErr != nil {
		log.Fatal(txErr)
	}
}

func LoadFileDumpStats(fileId string) {
	f := LoadFile(fileId)
	if f == nil {
		fmt.Println("File not found:", fileId)
		return
	}

	fmt.Printf("Title: %s \t Last Mod: %s \n", f.Title, f.ModifiedDate)
}
