package google

import (
	"log"
	"net/http"
	"time"

	activity "google.golang.org/api/appsactivity/v1"
	drive "google.golang.org/api/drive/v2"
	oauth "google.golang.org/api/oauth2/v2"
)

var (
	loginClient   *http.Client
	oauthSvc      *oauth.Service
	drvSvc        *drive.Service
	actSvc        *activity.Service
	driveThrottle <-chan time.Time
)

func init() {
	rate := time.Second / 10
	driveThrottle = time.Tick(rate)
}

func GetClientScope() []string {
	return []string{
		activity.ActivityScope,
		drive.DriveReadonlyScope,
		oauth.PlusMeScope,
		oauth.UserinfoEmailScope}
}

func setupClients(client *http.Client) {
	var err error

	loginClient = client

	oauthSvc, err = oauth.New(client)
	if err != nil {
		log.Fatalf("Unable to create OAuth service: %v", err)
	}

	drvSvc, err = drive.New(client)
	if err != nil {
		log.Fatalf("Unable to create Drive service: %v", err)
	}

	actSvc, err = appsactivity.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve appsactivity Client %v", err)
	}

}
