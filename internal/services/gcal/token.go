package gcal

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
)

// tokenSavingSource wraps an oauth2.TokenSource and automatically saves
// refreshed tokens to disk
type tokenSavingSource struct {
	source    oauth2.TokenSource
	tokenPath string
	lastToken *oauth2.Token
}

// Token returns a valid token, refreshing if necessary and saving to disk
func (t *tokenSavingSource) Token() (*oauth2.Token, error) {
	token, err := t.source.Token()
	if err != nil {
		return nil, err
	}

	// If this is a new token (different access token), save it
	if t.lastToken == nil || t.lastToken.AccessToken != token.AccessToken {
		if saveErr := saveToken(t.tokenPath, token); saveErr != nil {
			// Log the error but don't fail the request
			fmt.Fprintf(os.Stderr, "Warning: failed to save refreshed token: %v\n", saveErr)
		}
		t.lastToken = token
	}

	return token, nil
}

// saveToken saves the OAuth2 token to a file
func saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to save token: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(token)
}
