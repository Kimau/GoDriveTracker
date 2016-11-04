package main

import (
	"json"

	"github.com/ChimeraCoder/anaconda"
)

type ClientSecret struct {
	Key    string `json:"key"`
	Secret string `json:"secret"`

	AccessToken  string `json:"access_token"`
	AccessSecret string `json:"access_token_secret"`
}

func loadTwitterSecret(filename string) (*ClientSecret, error) {
	jsonBlob, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cs ClientSecret
	err = json.Unmarshal(jsonBlob, &cs)
	if err != nil {
		return nil, err
	}

	return &cs, nil
}

func startTwitterAPI() (*anaconda.TwitterApi, error) {
	secret, err := loadTwitterSecret("_twitter.json")
	if err != nil {
		log.Fatalln("Secret Missing: %s", err)
		return nil, err
	}

	anaconda.SetConsumerKey(secret.Key)
	anaconda.SetConsumerSecret(secret.Secret)
	api := anaconda.NewTwitterApi(secret.AccessToken, secret.AccessSecret)
	return api, nil
}
