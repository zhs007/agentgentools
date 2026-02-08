package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const maxDownloadBytes = 2 << 20 // 2 MiB

type hostInput struct {
	WasmPath     string
	ReadURL      string
	AllowURLs    []string
	OutputDir    string
	OutputFile   string
	DreamType    string
	DreamPhase   string
	DreamOutcome string
	DreamTags    string
	DreamMood    string
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func parseArgs(args []string) (hostInput, error) {
	var in hostInput
	var allowRaw string

	fs := flag.NewFlagSet("wasmrunner", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&in.WasmPath, "wasm", "", "path to wasm module")
	fs.StringVar(&in.ReadURL, "read-url", "", "https url to fetch as input")
	fs.StringVar(&allowRaw, "allow-urls", "", "comma separated exact allowed https urls")
	fs.StringVar(&in.OutputDir, "output-dir", "", "host output directory mounted to wasm")
	fs.StringVar(&in.OutputFile, "output-file", "result.json", "output file name under /out")
	fs.StringVar(&in.DreamType, "type", "", "dreamcard --type")
	fs.StringVar(&in.DreamPhase, "phase", "", "dreamcard --phase")
	fs.StringVar(&in.DreamOutcome, "outcome", "", "dreamcard --outcome")
	fs.StringVar(&in.DreamTags, "tags", "", "dreamcard --tags")
	fs.StringVar(&in.DreamMood, "mood", "", "dreamcard --mood")

	if err := fs.Parse(args); err != nil {
		return hostInput{}, err
	}

	in.AllowURLs = parseCSV(allowRaw)
	if in.WasmPath == "" {
		return hostInput{}, errors.New("--wasm is required")
	}
	if in.ReadURL == "" {
		return hostInput{}, errors.New("--read-url is required")
	}
	if len(in.AllowURLs) == 0 {
		return hostInput{}, errors.New("--allow-urls must not be empty")
	}
	if in.OutputDir == "" {
		return hostInput{}, errors.New("--output-dir is required")
	}

	return in, nil
}

func isAllowedHTTPS(target string, allow []string) error {
	if !strings.HasPrefix(strings.ToLower(target), "https://") {
		return fmt.Errorf("read-url must be https: %s", target)
	}
	for _, a := range allow {
		if target == a {
			return nil
		}
	}
	return fmt.Errorf("read-url not in allow list: %s", target)
}

func downloadText(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed status=%d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxDownloadBytes+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(b) > maxDownloadBytes {
		return nil, fmt.Errorf("download too large, max=%d bytes", maxDownloadBytes)
	}
	return b, nil
}

func runWasmWithInput(ctx context.Context, in hostInput, downloaded []byte) ([]byte, []byte, error) {
	outDir, err := filepath.Abs(in.OutputDir)
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, nil, err
	}

	inDir, err := os.MkdirTemp("", "wasmrunner-in-*")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(inDir)

	inFileHostPath := filepath.Join(inDir, "input.txt")
	if err := os.WriteFile(inFileHostPath, downloaded, 0o600); err != nil {
		return nil, nil, err
	}

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		return nil, nil, err
	}

	wasmBytes, err := os.ReadFile(in.WasmPath)
	if err != nil {
		return nil, nil, err
	}

	compiled, err := r.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, nil, err
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	moduleCfg := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stderr).
		WithArgs(
			"tool",
			"--text-file=/in/input.txt",
			"--out-file=/out/"+in.OutputFile,
			"--type="+in.DreamType,
			"--phase="+in.DreamPhase,
			"--outcome="+in.DreamOutcome,
			"--tags="+in.DreamTags,
			"--mood="+in.DreamMood,
		)

	fsCfg := wazero.NewFSConfig().
		WithDirMount(inDir, "/in").
		WithDirMount(outDir, "/out")
	moduleCfg = moduleCfg.WithFSConfig(fsCfg)

	if _, err := r.InstantiateModule(ctx, compiled, moduleCfg); err != nil {
		return stdout.Bytes(), stderr.Bytes(), err
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}

func main() {
	in, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse args failed:", err)
		os.Exit(1)
	}

	if err := isAllowedHTTPS(in.ReadURL, in.AllowURLs); err != nil {
		fmt.Fprintln(os.Stderr, "permission denied:", err)
		os.Exit(1)
	}

	downloaded, err := downloadText(in.ReadURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "download failed:", err)
		os.Exit(1)
	}

	stdout, stderr, err := runWasmWithInput(context.Background(), in, downloaded)
	if len(stdout) > 0 {
		_, _ = os.Stdout.Write(stdout)
	}
	if len(stderr) > 0 {
		_, _ = os.Stderr.Write(stderr)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "run wasm failed:", err)
		os.Exit(1)
	}
}
