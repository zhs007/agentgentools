package main

import "testing"

func TestParseCSV(t *testing.T) {
	got := parseCSV("a, b,, c")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("unexpected csv parse: %#v", got)
	}
}

func TestIsAllowedHTTPS(t *testing.T) {
	allow := []string{"https://example.com/a.txt"}

	if err := isAllowedHTTPS("http://example.com/a.txt", allow); err == nil {
		t.Fatalf("expected non-https to be denied")
	}
	if err := isAllowedHTTPS("https://example.com/b.txt", allow); err == nil {
		t.Fatalf("expected url not in list to be denied")
	}
	if err := isAllowedHTTPS("https://example.com/a.txt", allow); err != nil {
		t.Fatalf("expected allowed url, got err=%v", err)
	}
}

func TestParseArgs_Required(t *testing.T) {
	_, err := parseArgs([]string{})
	if err == nil {
		t.Fatalf("expected required args error")
	}
}

func TestParseArgs_OK(t *testing.T) {
	in, err := parseArgs([]string{
		"--wasm=./dreamcard.wasm",
		"--read-url=https://example.com/in.txt",
		"--allow-urls=https://example.com/in.txt",
		"--output-dir=./out",
	})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if in.WasmPath != "./dreamcard.wasm" || in.ReadURL != "https://example.com/in.txt" || in.OutputDir != "./out" {
		t.Fatalf("unexpected parsed input: %#v", in)
	}
	if len(in.AllowURLs) != 1 || in.AllowURLs[0] != "https://example.com/in.txt" {
		t.Fatalf("unexpected allow list: %#v", in.AllowURLs)
	}
}
