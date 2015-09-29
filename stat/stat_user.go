package stat

import (
	"fmt"
)

type UserStat struct {
	UpdateDate string `json:"UpdateDate"`
	Token      []byte `json:Token`
	Email      string `json:Email`
	UserID     string `json:Id`
}

func (usr *UserStat) String() string {
	return fmt.Sprintf("[%s] %s last updated on %s (TOKEN HIDDEN)", usr.UserID, usr.Email, usr.UpdateDate)
}
