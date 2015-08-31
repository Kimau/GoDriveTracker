package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	drive "google.golang.org/api/drive/v2"
	urlshortener "google.golang.org/api/urlshortener/v1"
)

const (
	mimeDoc string = "application/vnd.google-apps.document"
)

var (
	urlSvc *urlshortener.Service
	drvSvc *drive.Service
)

func init() {
	ClientScopes = append(ClientScopes, urlshortener.UrlshortenerScope, drive.DriveReadonlyScope)
	commandFuncs["getSortDocList"] = GetSortDriveList
}

func setupClients(client *http.Client) {
	var err error

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

	for i, v := range files {
		fmt.Printf("%6d: %s \t[%s]\t[%s]\n", i, v.Title, v.ModifiedByMeDate, v.ModifiedDate)
	}

	var revs []*drive.Revision

	{
		var err error
		lastFile := files[len(files)-1]
		revs, err = AllRevisions(lastFile.Id)

		if err != nil {
			log.Fatalln("Failed to get File Revisions", err)
			return err
		}
	}

	for i, v := range revs {
		fmt.Printf("%6d: \t[%s]\t[%s]\n", i, v.ModifiedDate, v.LastModifyingUserName)
	}

	return nil
}
