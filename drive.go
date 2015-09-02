package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	drive "google.golang.org/api/drive/v2"
	urlshortener "google.golang.org/api/urlshortener/v1"
)

const (
	mimeDoc string = "application/vnd.google-apps.document"
)

var (
	loginClient   *http.Client
	urlSvc        *urlshortener.Service
	drvSvc        *drive.Service
	driveThrottle <-chan time.Time
)

func init() {
	ClientScopes = append(ClientScopes, urlshortener.UrlshortenerScope, drive.DriveReadonlyScope)
	commandFuncs["doclist"] = GetSortDriveList

	rate := time.Second / 10
	driveThrottle = time.Tick(rate)
}

func setupClients(client *http.Client) {
	var err error

	loginClient = client

	urlSvc, err = urlshortener.New(client)
	if err != nil {
		log.Fatalf("Unable to create UrlShortener service: %v", err)
	}

	drvSvc, err = drive.New(client)
	if err != nil {
		log.Fatalf("Unable to create Drive service: %v", err)
	}
}

func getShortUrlDetail(shortUrl string) *urlshortener.Url {
	url, err := urlSvc.Url.Get(shortUrl).Do()
	if err != nil {
		log.Fatalf("URL Get: %v", err)
		return nil
	}
	fmt.Printf("Lookup of %s: %s\n", url, url.LongUrl)
	return url
}

func shortenUrl(longUrl string) *urlshortener.Url {
	url, err := urlSvc.Url.Insert(&urlshortener.Url{
		Kind:    "urlshortener#url", // Not really needed
		LongUrl: longUrl,
	}).Do()
	if err != nil {
		log.Fatalf("URL Insert: %v", err)
		return nil
	}
	fmt.Printf("Shortened %s => %s\n", url, url.Id)
	return url
}

// AllRevisions fetches all revisions for a given file
func AllRevisions(fileId string) ([]*drive.Revision, error) {
	<-driveThrottle // rate Limit
	r, err := drvSvc.Revisions.List(fileId).Do()
	if err != nil {
		fmt.Printf("An error occurred: %v\n", err)
		return nil, err
	}
	return r.Items, nil
}

// AllFiles fetches and displays all files
func AllFiles(query string) ([]*drive.File, error) {
	var fs []*drive.File
	pageToken := ""
	count := 0
	for {
		log.Println("Getting page of file listing", count)
		count = count + 1

		q := drvSvc.Files.List()
		q.Spaces("drive") // Only get drive (not 'appDataFolder' 'photos')
		q.Q(query)

		// If we have a pageToken set, apply it to the query
		if pageToken != "" {
			q = q.PageToken(pageToken)
		}

		<-driveThrottle // rate Limit
		r, err := q.Do()
		if err != nil {
			fmt.Printf("An error occurred: %v\n", err)
			return fs, err
		}
		fs = append(fs, r.Items...)
		pageToken = r.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return fs, nil
}

// Sort by Modified Date and Type
type ByTypeThenModMeDesc []*drive.File

func (a ByTypeThenModMeDesc) Len() int      { return len(a) }
func (a ByTypeThenModMeDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTypeThenModMeDesc) Less(i, j int) bool {
	return (a[i].MimeType < a[j].MimeType) ||
		((a[i].MimeType == a[j].MimeType) && ((a[i].ModifiedByMeDate < a[j].ModifiedByMeDate) ||
			((a[i].ModifiedByMeDate == a[j].ModifiedByMeDate) && (a[i].ModifiedDate < a[j].ModifiedDate))))
}

func GetFileRevsWriteDB(file *drive.File) error {
	revLists, err := AllRevisions(file.Id)

	for _, rev := range revLists {
		go WriteRevision(file.Id, rev)
	}

	if err != nil {
		log.Fatalln("Failed to get File Revisions", err)
		return err
	}

	return nil
}

func GetSortDriveList() error {
	var files []*drive.File
	{
		var err error
		files, err = AllFiles("mimeType = 'application/vnd.google-apps.document'")
		if err != nil {
			log.Fatalln("Failed to get File List", err)
			return err
		}
	}

	sort.Sort(ByTypeThenModMeDesc(files))

	var wg sync.WaitGroup

	for _, v := range files {
		wg.Add(1)
		go func(f *drive.File) {
			WriteFile(f)
			GetFileRevsWriteDB(f)
			wg.Done()
		}(v)
	}
	// Waiting on Writes
	fmt.Println("Waiting on Web Requests...")
	wg.Wait()

	return nil
}

func LoadFileDumpStats(fileId string) {
	f := LoadFile(fileId)
	if f == nil {
		fmt.Println("File not found:", fileId)
		return
	}

	fmt.Printf("%3d \tTitle: %s \n\t Last Mod: %s \n", f.Version, f.Title, f.ModifiedDate)

	<-driveThrottle // Rate Limit
	r, e := loginClient.Get(f.ExportLinks["text/plain"])
	if e != nil {
		log.Println("Failed to get text file", e.Error())
	} else {

		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		bodyStr := buf.String()
		wCount, total := WordCount(bodyStr)
		fmt.Printf("Word Count: %d \n Different Words: %d \n", total, len(wCount))

		/*
			fmt.Println("\n=========================================\n")

			fmt.Print(bodyStr)
			fmt.Println("\n=========================================\n")

			for k, v := range wCount {
				fmt.Println(k, ":", v)
			}*/
	}

	// Attempt to get revisions
	for rev := LoadNextRevision(fileId, ""); rev != nil; rev = LoadNextRevision(fileId, rev.Id) {
		<-driveThrottle // Rate Limit
		r, e := loginClient.Get(rev.ExportLinks["text/plain"])
		if e != nil {
			log.Println("Failed to get text file", e.Error())
			continue
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		bodyStr := buf.String()
		wCount, total := WordCount(bodyStr)
		fmt.Printf("REV: %s %s \n Word Count: %d \n Different Words: %d \n", rev.Id, rev.ModifiedDate, total, len(wCount))
	}

}

func GenerateStatsFile(file *drive.File) {
	dStat := DocStat{FileId: file.Id, LastMod: file.ModifiedDate}

	for rev := LoadNextRevision(file.Id, ""); rev != nil; rev = LoadNextRevision(file.Id, rev.Id) {
		<-driveThrottle // Rate Limit
		r, e := loginClient.Get(rev.ExportLinks["text/plain"])
		if e != nil {
			log.Println("Failed to get text file", e.Error())
			continue
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		bodyStr := buf.String()
		wCount, wTotal := WordCount(bodyStr)

		revStat := RevStat{RevId: rev.Id, WordCount: wTotal, ModDate: rev.ModifiedDate}
		for k, v := range wCount {
			revStat.WordFreq = append(revStat.WordFreq, WordPair{k, v})
		}
		sort.Sort(WordPairByVol(revStat.WordFreq))
		dStat.RevList = append(dStat.RevList, revStat)

	}

	WriteStats(&dStat)
	log.Println("Stats File Generated:", file.Title, file.Id)
}
