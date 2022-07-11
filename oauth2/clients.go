package oauth2

import "regexp"

type Client struct {
	RedirectURIPattern string `json:"redirect_uri_pattern,omitempty"`
}

type Clients map[string]Client

func (c Clients) ClientsMatchingRedirectURI(uri string) []string {
	var matches = make([]string, 0, len(c))
	for clientID, client := range c {
		if regexp.MustCompile(client.RedirectURIPattern).MatchString(uri) {
			matches = append(matches, clientID)
		}
	}
	return matches
}
