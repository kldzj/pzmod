package steam

import (
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
)

const (
	FileTypeMod        = 0
	FileTypeCollection = 2
)

var steamApiKey string = ""

func SetApiKey(key string) {
	steamApiKey = key
}

func getApiKey() string {
	return steamApiKey
}

func newHttpClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func constructSteamApiUrl(path string) (*url.URL, *url.Values) {
	url, err := url.Parse("https://api.steampowered.com" + path)
	if err != nil {
		cobra.CheckErr(err)
	}

	query := url.Query()
	query.Add("key", getApiKey())

	return url, &query
}
