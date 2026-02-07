package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mailru/easyjson/jwriter"
)

type Input struct {
	Text    string
	Type    string
	Phase   string
	Outcome string
	Tags    []string
	Mood    string
}

func (in *Input) MarshalEasyJSON(w *jwriter.Writer) {
	w.RawByte('{')
	w.RawString(`"text":`)
	w.String(in.Text)
	w.RawString(`,"type":`)
	w.String(in.Type)
	w.RawString(`,"phase":`)
	w.String(in.Phase)
	w.RawString(`,"outcome":`)
	w.String(in.Outcome)
	w.RawString(`,"tags":[`)
	for i := range in.Tags {
		if i > 0 {
			w.RawByte(',')
		}
		w.String(in.Tags[i])
	}
	w.RawString(`],"mood":`)
	w.String(in.Mood)
	w.RawByte('}')
}

func parseTags(raw string) []string {
	if raw == "" {
		return []string{}
	}

	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag == "" {
			continue
		}
		tags = append(tags, tag)
	}
	return tags
}

func buildInputFromArgs(args []string) (Input, error) {
	var in Input
	fs := flag.NewFlagSet("dreamcard", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var tagsRaw string
	fs.StringVar(&in.Text, "text", "", "text content")
	fs.StringVar(&in.Type, "type", "", "type value")
	fs.StringVar(&in.Phase, "phase", "", "phase value")
	fs.StringVar(&in.Outcome, "outcome", "", "outcome value")
	fs.StringVar(&tagsRaw, "tags", "", "comma separated tags")
	fs.StringVar(&in.Mood, "mood", "", "mood value")

	if err := fs.Parse(args); err != nil {
		return Input{}, err
	}
	in.Tags = parseTags(tagsRaw)
	return in, nil
}

func process(in Input) ([]byte, error) {
	var w jwriter.Writer
	in.MarshalEasyJSON(&w)
	return w.Buffer.BuildBytes(), nil
}

func main() {
	in, err := buildInputFromArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse args failed:", err)
		os.Exit(1)
	}
	out, err := process(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "process failed:", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		fmt.Fprintln(os.Stderr, "write stdout failed:", err)
		os.Exit(1)
	}
}
