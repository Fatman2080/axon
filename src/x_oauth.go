package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type XOAuthConfig struct {
	ClientID           string
	ClientSecret       string
	RedirectURL        string
	FrontendSuccessURL string
	FrontendFailureURL string
	Scopes             string
	AuthorizeURL       string
	TokenURL           string
	UserInfoURL        string
}

type XOAuthUser struct {
	ID              string
	Username        string
	Name            string
	ProfileImageURL string
}

func (c XOAuthConfig) Enabled() bool {
	return strings.TrimSpace(c.ClientID) != "" && strings.TrimSpace(c.RedirectURL) != ""
}

func randomURLSafeString(rawBytes int) (string, error) {
	buf := make([]byte, rawBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func buildCodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func buildRedirectURL(base string, params map[string]string) string {
	target := strings.TrimSpace(base)
	if target == "" {
		target = "/"
	}
	u, err := url.Parse(target)
	if err != nil {
		u, _ = url.Parse("/")
	}
	q := u.Query()
	for key, value := range params {
		if strings.TrimSpace(value) == "" {
			continue
		}
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *Server) buildXAuthorizeURL(state string, codeChallenge string) string {
	base := strings.TrimSpace(s.xOAuth.AuthorizeURL)
	if base == "" {
		base = "https://twitter.com/i/oauth2/authorize"
	}
	scopes := strings.TrimSpace(s.xOAuth.Scopes)
	if scopes == "" {
		scopes = "users.read tweet.read offline.access"
	}
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", s.xOAuth.ClientID)
	values.Set("redirect_uri", s.xOAuth.RedirectURL)
	values.Set("scope", scopes)
	values.Set("state", state)
	values.Set("code_challenge", codeChallenge)
	values.Set("code_challenge_method", "S256")
	authorizeURL := base + "?" + values.Encode()
	logInfo("oauth", "x oauth authorize redirect_uri=%s", s.xOAuth.RedirectURL)
	return authorizeURL
}

func (s *Server) exchangeXOAuthToken(code string, codeVerifier string) (string, error) {
	tokenURL := strings.TrimSpace(s.xOAuth.TokenURL)
	if tokenURL == "" {
		tokenURL = "https://api.twitter.com/2/oauth2/token"
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", strings.TrimSpace(code))
	form.Set("redirect_uri", s.xOAuth.RedirectURL)
	form.Set("code_verifier", strings.TrimSpace(codeVerifier))

	isConfidential := strings.TrimSpace(s.xOAuth.ClientSecret) != ""
	if !isConfidential {
		// Public client: client_id in form body
		form.Set("client_id", s.xOAuth.ClientID)
	}

	logInfo("oauth", "x oauth token exchange redirect_uri=%s confidential=%v", s.xOAuth.RedirectURL, isConfidential)

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if isConfidential {
		// Confidential client: Basic auth only (no client_id in form body per X OAuth 2.0 spec)
		cred := base64.StdEncoding.EncodeToString([]byte(s.xOAuth.ClientID + ":" + s.xOAuth.ClientSecret))
		req.Header.Set("Authorization", "Basic "+cred)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("x token endpoint status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	tokenResp := struct {
		AccessToken string `json:"access_token"`
	}{}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", errors.New("empty access token")
	}
	return tokenResp.AccessToken, nil
}

func (s *Server) fetchXOAuthUser(accessToken string) (XOAuthUser, error) {
	userInfoURL := strings.TrimSpace(s.xOAuth.UserInfoURL)
	if userInfoURL == "" {
		userInfoURL = "https://api.twitter.com/2/users/me"
	}

	u, err := url.Parse(userInfoURL)
	if err != nil {
		return XOAuthUser{}, err
	}
	q := u.Query()
	q.Set("user.fields", "profile_image_url,username,name")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), bytes.NewReader(nil))
	if err != nil {
		return XOAuthUser{}, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return XOAuthUser{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode >= 400 {
		return XOAuthUser{}, fmt.Errorf("x userinfo endpoint status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	userResp := struct {
		Data struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			Username        string `json:"username"`
			ProfileImageURL string `json:"profile_image_url"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(body, &userResp); err != nil {
		return XOAuthUser{}, err
	}
	if strings.TrimSpace(userResp.Data.ID) == "" {
		return XOAuthUser{}, errors.New("x user id is empty")
	}
	return XOAuthUser{
		ID:              userResp.Data.ID,
		Name:            userResp.Data.Name,
		Username:        userResp.Data.Username,
		ProfileImageURL: userResp.Data.ProfileImageURL,
	}, nil
}
