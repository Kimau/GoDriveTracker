package google

import (
	"fmt"
	"net/http"
	// oauth "google.golang.org/api/oauth2/v2"
)

func GetIdentity() (string, error) {
	tokenCall := oauthSvc.Tokeninfo()
	tokenCall.AccessToken(Token.AccessToken)
	token, err := tokenCall.Do()
	if err != nil {
		return "", err
	}

	outStr := fmt.Sprintf("AccessType: %s, \n Email: %s, \n UserId: %s, \n", token.AccessType, token.Email, token.UserId)

	return outStr, nil
}

func GetAuth(getUrl string) (resp *http.Response, err error) {
	<-driveThrottle // Rate Limit
	r, e := loginClient.Get(getUrl)

	return r, e
}
