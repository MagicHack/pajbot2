package web

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"
)

// Commit github json to go
type Commit struct {
	ID        string    `json:"id"`
	TreeID    string    `json:"tree_id"`
	Distinct  bool      `json:"distinct"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	URL       string    `json:"url"`
	Author    struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Username string `json:"username"`
	} `json:"author"`
	Committer struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Username string `json:"username"`
	} `json:"committer"`
}

// RepositoryData xD
type RepositoryData struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"owner"`
	HTMLURL string `json:"html_url"`
	URL     string `json:"url"`
}

// PushHookResponse github json to go
type PushHookResponse struct {
	Commits    []Commit       `json:"commits"`
	HeadCommit Commit         `json:"head_commit"`
	Repository RepositoryData `json:"repository"`
	Pusher     struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"pusher"`
	Sender struct {
		Login string `json:"login"`
		ID    int    `json:"id"`
		URL   string `json:"url"`
	} `json:"sender"`
}

func signBody(secret, body []byte) []byte {
	computed := hmac.New(sha1.New, secret)
	computed.Write(body)
	return []byte(computed.Sum(nil))
}

func verifySignature(secretString string, signature string, body []byte) bool {
	const signaturePrefix = "sha1="
	const signatureLength = 45

	if len(signature) != signatureLength || !strings.HasPrefix(signature, signaturePrefix) {
		return false
	}

	secret := []byte(secretString)
	actual := make([]byte, 20)
	hex.Decode(actual, []byte(signature[5:]))

	return hmac.Equal(signBody(secret, body), actual)
}
