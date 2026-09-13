package utils

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

const CanvasFieldLimit = 50 * 1024

var forbiddenCanvasHTML = regexp.MustCompile(`(?i)<\s*(?:script|iframe|object|embed|meta|base|link|form|a|area)(?:\s|/?>)`)

var forbiddenCanvasJavaScript = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:fetch|xmlhttprequest|websocket|eventsource|sendbeacon|indexeddb|broadcastchannel|webtransport|rtcpeerconnection|sharedworker|worker|serviceworker|navigator)\b`),
	regexp.MustCompile(`(?i)\b(?:location|parent|top|opener|postmessage|localstorage|sessionstorage|caches|defaultview)\b`),
	regexp.MustCompile(`(?i)\bdocument\s*\.\s*(?:cookie|domain)\b`),
	regexp.MustCompile(`(?i)\b(?:window|self|globalthis|document)\s*\[`),
}

type CanvasPayload struct {
	Version             int    `json:"version"`
	DescriptionMarkdown string `json:"descriptionMarkdown"`
	HTML                string `json:"html"`
	CSS                 string `json:"css"`
	JS                  string `json:"js"`
}

func ValidateCanvas(payload CanvasPayload) error {
	if payload.Version != 1 {
		return errors.New("unsupported canvas version")
	}
	for _, field := range []string{payload.HTML, payload.CSS, payload.JS} {
		if len([]byte(field)) > CanvasFieldLimit {
			return errors.New("canvas HTML, CSS and JavaScript must each be at most 50 KiB")
		}
	}
	if len([]byte(payload.DescriptionMarkdown)) > CanvasFieldLimit {
		return errors.New("canvas description must be at most 50 KiB")
	}
	if forbiddenCanvasHTML.MatchString(payload.HTML) {
		return errors.New("scripts, navigation, forms and nested browsing contexts are not allowed in canvas HTML")
	}
	for _, pattern := range forbiddenCanvasJavaScript {
		if pattern.MatchString(payload.JS) {
			return errors.New("canvas JavaScript requests a forbidden network, storage, identity or navigation capability")
		}
	}
	return nil
}

func EncodeCanvas(payload CanvasPayload) (string, error) {
	if err := ValidateCanvas(payload); err != nil {
		return "", err
	}
	data, err := json.Marshal(payload)
	return string(data), err
}

func DecodeCanvas(raw string) (*CanvasPayload, error) {
	var payload CanvasPayload
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	if err := ValidateCanvas(payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
