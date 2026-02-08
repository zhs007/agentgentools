package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseTags(t *testing.T) {
	got := parseTags("cat, horror,meta,, ")
	if len(got) != 3 || got[0] != "cat" || got[1] != "horror" || got[2] != "meta" {
		t.Fatalf("unexpected tags: %#v", got)
	}
}

func TestBuildInputFromArgs(t *testing.T) {
	in, err := buildInputFromArgs([]string{
		"--text=hello",
		"--type=weird",
		"--phase=high",
		"--outcome=win",
		"--tags=cat,horror,meta",
		"--mood=dark",
	})
	if err != nil {
		t.Fatalf("buildInputFromArgs failed: %v", err)
	}

	if in.Text != "hello" || in.Type != "weird" || in.Phase != "high" || in.Outcome != "win" || in.Mood != "dark" {
		t.Fatalf("unexpected scalar fields: %#v", in)
	}
	if len(in.Tags) != 3 || in.Tags[0] != "cat" || in.Tags[1] != "horror" || in.Tags[2] != "meta" {
		t.Fatalf("unexpected tags: %#v", in.Tags)
	}
}

func TestBuildInputFromArgs_InvalidFlag(t *testing.T) {
	_, err := buildInputFromArgs([]string{"--bad=1"})
	if err == nil {
		t.Fatal("expected error for invalid flag, got nil")
	}
}

func TestProcess_Sample(t *testing.T) {
	in := Input{
		Text:    "The stray cat you fed looks at you. 'Wake up, Jack,' it says in your father's voice. 'You're in a coma.'",
		Type:    "weird",
		Phase:   "high",
		Outcome: "win",
		Tags:    []string{"cat", "horror", "meta"},
		Mood:    "dark",
	}

	out, err := process(in)
	if err != nil {
		t.Fatalf("process failed: %v", err)
	}

	want := `{"text":"The stray cat you fed looks at you. 'Wake up, Jack,' it says in your father's voice. 'You're in a coma.'","type":"weird","phase":"high","outcome":"win","tags":["cat","horror","meta"],"mood":"dark"}`
	if string(out) != want {
		t.Fatalf("unexpected output\nwant: %s\ngot:  %s", want, string(out))
	}
}

func TestMain_Args_E2E(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--text=hello", "--type=weird", "--phase=mid", "--outcome=win", "--tags=cat", "--mood=dark")
	cmd.Dir = "."

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("run failed: %v, stderr: %s", err, stderr.String())
	}

	want := `{"text":"hello","type":"weird","phase":"mid","outcome":"win","tags":["cat"],"mood":"dark"}`
	if stdout.String() != want {
		t.Fatalf("unexpected stdout\nwant: %s\ngot:  %s", want, stdout.String())
	}
}

func TestBuildInputFromArgs_TextFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "in.txt")
	if err := os.WriteFile(p, []byte("from-file"), 0o600); err != nil {
		t.Fatalf("write input file failed: %v", err)
	}

	in, err := buildInputFromArgs([]string{"--text-file=" + p})
	if err != nil {
		t.Fatalf("buildInputFromArgs failed: %v", err)
	}
	if in.Text != "from-file" {
		t.Fatalf("unexpected text from file: %q", in.Text)
	}
}

func TestMain_OutFile_E2E(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.json")

	cmd := exec.Command("go", "run", ".", "--text=hello", "--out-file="+outFile)
	cmd.Dir = "."

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("run failed: %v, stderr: %s", err, stderr.String())
	}

	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read out file failed: %v", err)
	}
	want := `{"text":"hello","type":"","phase":"","outcome":"","tags":[],"mood":""}`
	if string(b) != want {
		t.Fatalf("unexpected out-file content\nwant: %s\ngot:  %s", want, string(b))
	}
}
