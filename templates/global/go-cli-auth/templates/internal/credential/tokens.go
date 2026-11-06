package credential

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

type Tokens struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	TokenType    string    `json:"tokenType,omitempty"`
	Expiry       time.Time `json:"expiry"`
	Account      string    `json:"account,omitempty"`
}

func Logout(ctx context.Context, warnings io.Writer) (bool, error) {
	host, err := APIHost()
	if err != nil {
		return false, err
	}
	store, err := NewStore()
	if err != nil {
		return false, err
	}
	stored, err := store.Get(host)
	if errors.Is(err, ErrNotStored) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := revoke(ctx, stored.Value); err != nil {
		_, _ = fmt.Fprintf(warnings, "The issuer did not revoke the token: %v\n", err)
	}
	return store.Delete(host)
}

func revoke(ctx context.Context, stored string) error {
	var tokens Tokens
	if err := json.Unmarshal([]byte(stored), &tokens); err != nil {
		return err
	}
	found, err := discover(ctx)
	if err != nil {
		return err
	}
	return found.revoke(ctx, tokens)
}

func storedTokens(ctx context.Context) (Credential, error) {
	host, err := APIHost()
	if err != nil {
		return Credential{}, err
	}
	store, err := NewStore()
	if err != nil {
		return Credential{}, err
	}
	stored, err := store.Get(host)
	if err != nil {
		return Credential{}, err
	}
	var tokens Tokens
	if err := json.Unmarshal([]byte(stored.Value), &tokens); err != nil {
		return Credential{}, fmt.Errorf("the stored tokens for %s: %w", host, err)
	}
	if !tokens.oauth2Token().Valid() {
		if tokens, err = refresh(ctx, tokens, host); err != nil {
			return Credential{}, err
		}
		if stored.Source, err = save(store, host, tokens); err != nil {
			return Credential{}, err
		}
	}
	return Credential{Value: tokens.AccessToken, Source: stored.Source, Account: tokens.Account, Expiry: tokens.Expiry}, nil
}

func refresh(ctx context.Context, tokens Tokens, host string) (Tokens, error) {
	if tokens.RefreshToken == "" {
		return Tokens{}, fmt.Errorf(`the tokens for %s expired: run "%s auth login"`, host, name)
	}
	found, err := discover(ctx)
	if err != nil {
		return Tokens{}, err
	}
	config := found.config()
	token, err := config.TokenSource(withHTTPClient(ctx), tokens.oauth2Token()).Token()
	if err != nil {
		return Tokens{}, fmt.Errorf(`refresh the tokens for %s: %w: run "%s auth login"`, host, err, name)
	}
	return newTokens(token, tokens), nil
}

func save(store Store, host string, tokens Tokens) (string, error) {
	text, err := json.Marshal(tokens)
	if err != nil {
		return "", err
	}
	return store.Set(host, string(text))
}

func newTokens(token *oauth2.Token, previous Tokens) Tokens {
	tokens := Tokens{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
		Account:      previous.Account,
	}
	if idToken, ok := token.Extra("id_token").(string); ok {
		if account := accountOf(idToken); account != "" {
			tokens.Account = account
		}
	}
	return tokens
}

func (t Tokens) oauth2Token() *oauth2.Token {
	return &oauth2.Token{AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, TokenType: t.TokenType, Expiry: t.Expiry}
}

// the CLI only shows this claim to the user, so it does not check the signature of the ID token
func accountOf(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
		Subject           string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	for _, account := range []string{claims.Email, claims.PreferredUsername, claims.Subject} {
		if account != "" {
			return account
		}
	}
	return ""
}
