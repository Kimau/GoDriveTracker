package main

/*
import (
	database "./database"
	google "./google"
	stat "./stat"
	web "./web"
	drive "google.golang.org/api/drive/v2" // DO NOT LIKE THIS! Want to encapse this in google package
)

func SetupDatabase(wf *web.WebFace, db *database.StatTrackerDB) {
	wf.RedirectHandler = func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "Starting Server on %s", *addr)
	}

	cPage := make(chan int)
	google.AllFiles("mimeType = 'application/vnd.google-apps.document'", cPage)

	// TODO :: Loads
}
/**/
