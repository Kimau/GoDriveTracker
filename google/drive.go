package google

import (
	"fmt"
	"io/ioutil"
	"time"

	drive "google.golang.org/api/drive/v2"
)

const (
	MimeDoc string = "application/vnd.google-apps.document"
)

type RevBody struct {
	Rev     *drive.Revision
	ModTime time.Time
	Body    string
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
func AllFiles(query string, pageNum chan int) ([]*drive.File, error) {
	var fs []*drive.File
	pageToken := ""
	count := 0
	for {
		count = count + 1

		q := drvSvc.Files.List()
		q.Spaces("drive") // Only get drive (not 'appDataFolder' 'photos')
		q.Q(query)

		// If we have a pageToken set, apply it to the query
		if pageToken != "" {
			q = q.PageToken(pageToken)
		}

		pageNum <- count
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

	pageNum <- -1
	return fs, nil
}

func DownloadFileRev(fileId string, revId string) ([]byte, error) {
	<-driveThrottle // rate Limit
	r, err := loginClient.Get(fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/revisions/%s/", fileId, revId))

	if err != nil {
		return nil, err
	}

	body, rErr := ioutil.ReadAll(r.Body)
	if rErr != nil {
		return nil, rErr
	}

	return body, nil
}

func ExportRev(rev *drive.Revision) (string, error) {
	<-driveThrottle // rate Limit
	var ba []byte
	r, err := GetAuth(rev.ExportLinks["text/plain"])
	if err != nil {
		return "", err
	}

	ba, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	return string(ba), nil
}
