package utils

import (
	"strings"
	"testing"
)

func TestCanvasRoundTrip(t *testing.T) {
	want := CanvasPayload{Version: 1, DescriptionMarkdown: "hello", HTML: "<canvas></canvas>", CSS: "canvas{}", JS: "const x = 1"}
	raw, err := EncodeCanvas(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeCanvas(raw)
	if err != nil {
		t.Fatal(err)
	}
	if *got != want {
		t.Fatalf("got %#v want %#v", *got, want)
	}
}

func TestCanvasRejectsNestedExecutableMarkup(t *testing.T) {
	for _, html := range []string{
		"<script>alert(1)</script>",
		"<iframe src='x'></iframe>",
		"<object></object>",
		"<meta http-equiv='refresh' content='0;https://example.com'>",
		"<a href='https://example.com'>leave</a>",
	} {
		if err := ValidateCanvas(CanvasPayload{Version: 1, HTML: html}); err == nil {
			t.Fatalf("expected %q to fail", html)
		}
	}
}

func TestCanvasRejectsForbiddenBrowserCapabilities(t *testing.T) {
	for _, javascript := range []string{
		"fetch('https://example.com')",
		"localStorage.setItem('x', 'y')",
		"parent.postMessage('x', '*')",
		"location.href = 'https://example.com'",
		"globalThis['Web' + 'Socket']",
	} {
		if err := ValidateCanvas(CanvasPayload{Version: 1, JS: javascript}); err == nil {
			t.Fatalf("expected %q to fail", javascript)
		}
	}
}

func TestCanvasRejectsOversizedFieldsAndUnknownJSON(t *testing.T) {
	if err := ValidateCanvas(CanvasPayload{Version: 1, JS: strings.Repeat("x", CanvasFieldLimit+1)}); err == nil {
		t.Fatal("expected oversized JS to fail")
	}
	if _, err := DecodeCanvas(`{"version":1,"html":"","css":"","js":"","descriptionMarkdown":"","network":true}`); err == nil {
		t.Fatal("expected unknown field to fail")
	}
}
