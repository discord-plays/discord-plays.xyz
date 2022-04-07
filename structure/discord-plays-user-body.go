package structure

type DiscordPlaysUserBody struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Admin    bool   `json:"admin"`
}
