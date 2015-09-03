package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/boltdb/bolt"

	drive "google.golang.org/api/drive/v2"
)

var bucketDoc = []byte("doc")
var bucketRevs = []byte("revs")
var bucketDocStats = []byte("stats")
var bucketDaily = []byte("daily")
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

func WriteRevision(fileId string, rev *drive.Revision) {

	writeRevFunc := func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketRevs)
		if err != nil {
			return err
		}

		dat, e2 := json.Marshal(rev)

		e2 = bucket.Put([]byte(fileId+" "+rev.Id), dat)
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

func LoadNextFile(fileId string) *drive.File {
	var result drive.File

	seekKey := []byte(fileId)

	loadFunc := func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketDoc)
		if bucket == nil {
			log.Printf("Bucket %q not found!", bucketDoc)
			return errors.New("Bucket not found!")
		}

		c := bucket.Cursor()
		k, v := c.First()
		if len(fileId) > 0 {
			k, v = c.Seek(seekKey)
			k, v = c.Next()
		}

		if k == nil {
			return errors.New("No more Files")
		}

		errMarshal := json.Unmarshal(v, &result)
		if errMarshal != nil {
			log.Println("Unmarshal failed:", errMarshal)
			return errMarshal
		}
		return nil
	}

	// retrieve the data
	txErr := boltDB.View(loadFunc)
	if txErr != nil {
		// log.Fatalln(txErr)
		return nil
	}

	return &result
}

func LoadNextRevision(fileId string, revID string) *drive.Revision {
	var result drive.Revision

	seekKey := []byte(fileId + " " + revID)

	loadFunc := func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketRevs)
		if bucket == nil {
			log.Printf("Bucket %q not found!", bucketRevs)
			return errors.New("Bucket not found!")
		}

		c := bucket.Cursor()
		c.First()
		k, v := c.Seek(seekKey)
		if revID != "" {
			k, v = c.Next()
		}

		if k == nil {
			return errors.New("No more revisions")
		}
		kStr := string(k)

		if strings.HasPrefix(kStr, fileId) {
			errMarshal := json.Unmarshal(v, &result)
			if errMarshal != nil {
				log.Println("Unmarshal failed:", errMarshal)
				return errMarshal
			}
			return nil
		} else {
			return errors.New("No more revisions")
		}

		return nil
	}

	// retrieve the data
	txErr := boltDB.View(loadFunc)
	if txErr != nil {
		// log.Fatalln(txErr)
		return nil
	}

	return &result
}

func WriteFileStats(fStat *DocStat) {
	writeFunc := func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketDocStats)
		if err != nil {
			log.Println("Bucket failed:", err)
			return err
		}

		dat, eMarshal := json.Marshal(fStat)
		if eMarshal != nil {
			log.Println("Marhsal failed:", eMarshal)
			return eMarshal
		}

		ePut := bucket.Put([]byte(fStat.FileId), dat)
		if ePut != nil {
			log.Println("Put failed:", ePut)
			return ePut
		}

		return nil
	}

	// store some data
	txErr := boltDB.Update(writeFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}
}

func LoadFileStats(fileId string) *DocStat {
	var result DocStat

	loadFunc := func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketDocStats)
		if bucket == nil {
			log.Printf("Bucket %q not found!", bucketDocStats)
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
	txErr := boltDB.View(loadFunc)
	if txErr != nil {
		return nil
	}

	return &result
}

func WriteDailyStats(day *DailyStat) {
	writeFunc := func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketDaily)
		if err != nil {
			log.Println("Bucket failed:", err)
			return err
		}

		dat, eMarshal := json.Marshal(day)
		if eMarshal != nil {
			log.Println("Marhsal failed:", eMarshal)
			return eMarshal
		}

		ePut := bucket.Put([]byte(day.ModDate), dat)
		if ePut != nil {
			log.Println("Put failed:", ePut)
			return ePut
		}

		return nil
	}

	// store some data
	txErr := boltDB.Update(writeFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}
}

func LoadDailyStats(shortDate string) *DailyStat {
	var result DailyStat

	loadFunc := func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketDaily)
		if bucket == nil {
			log.Printf("Bucket %q not found!", bucketDaily)
			return errors.New("Bucket not found!")
		}

		dat := bucket.Get([]byte(shortDate))
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
	txErr := boltDB.View(loadFunc)
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
