package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	urlshortener "google.golang.org/api/urlshortener/v1"
)

func urlShortenerMain(client *http.Client, urls []string) {

	svc, err := urlshortener.New(client)
	if err != nil {
		log.Fatalf("Unable to create UrlShortener service: %v", err)
	}
	
	for _, u range urls {
		// short -> long
		if strings.HasPrefix(u, "http://goo.gl/") || strings.HasPrefix(u, "https://goo.gl/") {
			url, err := svc.Url.Get(urlstr).Do()
			if err != nil {
				log.Fatalf("URL Get: %v", err)
			}
			fmt.Printf("Lookup of %s: %s\n", u, url.LongUrl)
			return
		}
	
		// long -> short
		url, err := svc.Url.Insert(&urlshortener.Url{
			Kind:    "urlshortener#url", // Not really needed
			LongUrl: u,
		}).Do()
		if err != nil {
			log.Fatalf("URL Insert: %v", err)
		}
		fmt.Printf("Shortened %s => %s\n", u, url.Id)
	}
}
