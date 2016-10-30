package google

import (
	"fmt"

	activity "google.golang.org/api/appsactivity/v1"
	drive "google.golang.org/api/drive/v2"
)

// AllRevisions fetches all revisions for a given file
func ListActivities(reqPageSize int) ([]*activity.Activity, error) {
	<-driveThrottle // rate Limit

	r, err := actSvc.Activities.List().Source("drive.google.com").
		DriveAncestorId("root").PageSize(reqPageSize).Do()
	if err != nil {
		return nil, err
	}

	return r, nil
}
