package structure

import "time"

type DiscordMeBody struct {
	Id            string    `json:"id"`
	Username      string    `json:"username"`
	Discriminator string    `json:"discriminator"`
	Avatar        string    `json:"avatar"`
	LoggedInUntil time.Time `json:"-"`
}
