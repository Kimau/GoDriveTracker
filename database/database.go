package database

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/boltdb/bolt"

	stat "../stat"
	drive "google.golang.org/api/drive/v2" // DO NOT LIKE THIS! Want to encapse this in google package
)

var bucketUser = []byte("user")
var bucketDoc = []byte("doc")
var bucketRevs = []byte("revs")
var bucketDocStats = []byte("stats")
var bucketDaily = []byte("daily")

type StatTrackerDB struct {
	db *bolt.DB
}

func OpenDB(filename string) *StatTrackerDB {
	var err error

	dbPtr, err := bolt.Open(filename, 0600, nil)
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
	txErr := dbPtr.Update(writeFileFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}

	return &StatTrackerDB{db: dbPtr}
}

func (st *StatTrackerDB) CloseDB() {
	err := st.db.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func (st *StatTrackerDB) WriteFile(file *drive.File) {
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
	txErr := st.db.Update(writeFileFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}
}

func (st *StatTrackerDB) LoadFile(fileId string) *drive.File {
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
	txErr := st.db.View(loadFileFunc)
	if txErr != nil {
		return nil
	}

	return &result
}

func (st *StatTrackerDB) WriteRevision(fileId string, rev *drive.Revision) {

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

		//log.Println("Write Revision:", rev.Id)

		return nil
	}

	// store some data
	txErr := st.db.Update(writeRevFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}
}

func (st *StatTrackerDB) LoadNextFile(fileId string) *drive.File {
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
	txErr := st.db.View(loadFunc)
	if txErr != nil {
		// log.Fatalln(txErr)
		return nil
	}

	return &result
}

func (st *StatTrackerDB) LoadNextRevision(fileId string, revID string) *drive.Revision {
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
		// Unreachable
	}

	// retrieve the data
	txErr := st.db.View(loadFunc)
	if txErr != nil {
		// log.Fatalln(txErr)
		return nil
	}

	return &result
}

func (st *StatTrackerDB) LoadNextFileStat(fileId string) *stat.DocStat {
	var result stat.DocStat

	seekKey := []byte(fileId)

	loadFunc := func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketDocStats)
		if bucket == nil {
			log.Printf("Bucket %q not found!", bucketDocStats)
			return errors.New("Bucket not found!")
		}

		c := bucket.Cursor()
		k, v := c.First()
		if len(fileId) > 0 {
			k, v = c.Seek(seekKey)
			k, v = c.Next()
		}

		if k == nil {
			return errors.New("No more Doc Stats")
		}

		errMarshal := json.Unmarshal(v, &result)
		if errMarshal != nil {
			log.Println("Unmarshal failed:", errMarshal)
			return errMarshal
		}
		return nil
	}

	// retrieve the data
	txErr := st.db.View(loadFunc)
	if txErr != nil {
		// log.Fatalln(txErr)
		return nil
	}

	return &result
}

func (st *StatTrackerDB) LoadNextDailyStat(shortDate string) *stat.DailyStat {
	var result stat.DailyStat

	seekKey := []byte(shortDate)

	loadFunc := func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketDaily)
		if bucket == nil {
			log.Printf("Bucket %q not found!", bucketDaily)
			return errors.New("Bucket not found!")
		}

		c := bucket.Cursor()
		k, v := c.First()
		if len(shortDate) > 0 {
			k, v = c.Seek(seekKey)
			k, v = c.Next()
		}

		if k == nil {
			return errors.New("No more Faily Stats")
		}

		errMarshal := json.Unmarshal(v, &result)
		if errMarshal != nil {
			log.Println("Unmarshal failed:", errMarshal)
			return errMarshal
		}
		return nil
	}

	// retrieve the data
	txErr := st.db.View(loadFunc)
	if txErr != nil {
		// log.Fatalln(txErr)
		return nil
	}

	return &result
}

func (st *StatTrackerDB) WriteFileStats(fStat *stat.DocStat) {
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
	txErr := st.db.Update(writeFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}
}

func (st *StatTrackerDB) LoadFileStats(fileId string) *stat.DocStat {
	var result stat.DocStat

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
	txErr := st.db.View(loadFunc)
	if txErr != nil {
		return nil
	}

	return &result
}

func (st *StatTrackerDB) WriteDailyStats(day *stat.DailyStat) {
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
	txErr := st.db.Update(writeFunc)
	if txErr != nil {
		log.Fatal(txErr)
	}
}

func (st *StatTrackerDB) LoadDailyStats(shortDate string) *stat.DailyStat {
	var result stat.DailyStat

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
	txErr := st.db.View(loadFunc)
	if txErr != nil {
		return nil
	}

	return &result
}
