package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tukaelu/zgsync/internal/cli/testhelper"
	"github.com/tukaelu/zgsync/internal/zendesk"
	"gopkg.in/yaml.v3"
)

// magic bytes for image format detection
var (
	pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 13}
	jpegMagic = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F'}
	gif89aMagic = []byte("GIF89a")
	// WebP requires "RIFF????WEBP" — 12 bytes minimum so http.DetectContentType matches.
	webpMagic = append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), bytes.Repeat([]byte{0}, 4)...)
)

// fakeNetError implements net.Error so we can drive isRetryable's timeout branch.
type fakeNetError struct {
	msg     string
	timeout bool
}

func (e *fakeNetError) Error() string   { return e.msg }
func (e *fakeNetError) Timeout() bool   { return e.timeout }
func (e *fakeNetError) Temporary() bool { return false }

func TestWithMediaRetry_SucceedsImmediately(t *testing.T) {
	t.Parallel()
	calls := 0
	err := withMediaRetry(3, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestWithMediaRetry_SucceedsAfterRetry(t *testing.T) {
	t.Parallel()
	calls := 0
	err := withMediaRetry(3, func() error {
		calls++
		if calls < 3 {
			return &zendesk.HTTPError{StatusCode: 503}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil after retry success, got %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestWithMediaRetry_NonRetryableExitsImmediately(t *testing.T) {
	t.Parallel()
	calls := 0
	err := withMediaRetry(3, func() error {
		calls++
		return &zendesk.HTTPError{StatusCode: 400}
	})

	if calls != 1 {
		t.Errorf("calls = %d, want 1 (non-retryable must not retry)", calls)
	}
	var httpErr *zendesk.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *zendesk.HTTPError, got %T (%v)", err, err)
	}
	if httpErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400 (must surface the original status)", httpErr.StatusCode)
	}
	var exhausted *errRetriesExhausted
	if errors.As(err, &exhausted) {
		t.Errorf("non-retryable error must not be wrapped in errRetriesExhausted")
	}
}

func TestWithMediaRetry_Exhaustion(t *testing.T) {
	t.Parallel()
	calls := 0
	err := withMediaRetry(3, func() error {
		calls++
		return &zendesk.HTTPError{StatusCode: 503}
	})

	// 1 initial + 3 retries = 4 invocations.
	if calls != 4 {
		t.Errorf("calls = %d, want 4 (initial + 3 retries)", calls)
	}
	var exhausted *errRetriesExhausted
	if !errors.As(err, &exhausted) {
		t.Fatalf("expected *errRetriesExhausted, got %T (%v)", err, err)
	}
	if exhausted.retries != 3 {
		t.Errorf("retries = %d, want 3", exhausted.retries)
	}
	// Unwrap should expose the last underlying error.
	var httpErr *zendesk.HTTPError
	if !errors.As(exhausted, &httpErr) || httpErr.StatusCode != 503 {
		t.Errorf("Unwrap chain missing 503: %v", exhausted)
	}
}

func TestErrRetriesExhausted_ErrorMessage(t *testing.T) {
	t.Parallel()
	e := &errRetriesExhausted{retries: 3, lastErr: errors.New("boom")}
	if !strings.Contains(e.Error(), "gave up after 3 retries") {
		t.Errorf("Error() = %q, missing retry count phrase", e.Error())
	}
	if !strings.Contains(e.Error(), "boom") {
		t.Errorf("Error() = %q, missing wrapped cause", e.Error())
	}
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"500", &zendesk.HTTPError{StatusCode: 500}, true},
		{"502", &zendesk.HTTPError{StatusCode: 502}, true},
		{"503", &zendesk.HTTPError{StatusCode: 503}, true},
		{"599", &zendesk.HTTPError{StatusCode: 599}, true},
		{"400", &zendesk.HTTPError{StatusCode: 400}, false},
		{"401", &zendesk.HTTPError{StatusCode: 401}, false},
		{"404", &zendesk.HTTPError{StatusCode: 404}, false},
		{"422", &zendesk.HTTPError{StatusCode: 422}, false},
		{"499", &zendesk.HTTPError{StatusCode: 499}, false},
		{"wrapped 500 unwraps", fmt.Errorf("step 1: %w", &zendesk.HTTPError{StatusCode: 500}), true},
		{"wrapped 400 stays non-retryable", fmt.Errorf("step 1: %w", &zendesk.HTTPError{StatusCode: 400}), false},
		{"net.Error timeout=true", &fakeNetError{msg: "i/o timeout", timeout: true}, true},
		{"net.Error timeout=false", &fakeNetError{msg: "connection refused", timeout: false}, false},
		{"plain error", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isRetryable(tt.err); got != tt.want {
				t.Errorf("isRetryable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestDetectMediaMIMEType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ext      string
		contents []byte
		want     string
	}{
		{"png magic", ".png", pngMagic, "image/png"},
		{"jpeg magic", ".jpg", jpegMagic, "image/jpeg"},
		{"gif magic", ".gif", gif89aMagic, "image/gif"},
		{"webp magic", ".webp", webpMagic, "image/webp"},
		// Fallback paths: bytes don't match any image format, but the extension does.
		{"webp fallback by extension", ".webp", []byte("not really a webp file"), "image/webp"},
		{"png fallback by extension", ".png", []byte("plain text"), "image/png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "sample"+tt.ext)
			if err := os.WriteFile(path, tt.contents, 0o644); err != nil {
				t.Fatal(err)
			}
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = f.Close() })

			got, err := detectMediaMIMEType(f)
			if err != nil {
				t.Fatalf("detectMediaMIMEType: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}

			// File must be rewound so the caller can read from byte 0.
			pos, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				t.Fatal(err)
			}
			if pos != 0 {
				t.Errorf("file position after detection = %d, want 0", pos)
			}
		})
	}
}

// detectMediaMIMEType must NOT default to image/* for non-image files even
// when the extension is unknown — caller relies on the allowlist check.
func TestDetectMediaMIMEType_UnknownNonImageReturnsAsIs(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(path, []byte("just some plain text content here"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })

	got, err := detectMediaMIMEType(f)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(got, "image/") {
		t.Errorf("plain text should not be detected as image/*, got %q", got)
	}
}

func TestCwdRelative(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "images")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// EvalSymlinks because t.TempDir on macOS resolves /var → /private/var.
	resolvedTempDir, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(resolvedTempDir)

	tests := []struct {
		name string
		path string
		want string
	}{
		{"relative path in subdir", "images/foo.png", "images/foo.png"},
		{"absolute path inside cwd", filepath.Join(resolvedTempDir, "images", "foo.png"), "images/foo.png"},
		{"file in same dir", "foo.png", "foo.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cwdRelative(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// Files outside the CWD subtree must fall back to the base name so the
// Guide Media `name` field doesn't contain `..` segments.
func TestCwdRelative_OutsideCwdFallsBackToBaseName(t *testing.T) {
	cwd := t.TempDir()
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "remote.png")

	resolvedCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(resolvedCwd)

	got, err := cwdRelative(outsidePath)
	if err != nil {
		t.Fatal(err)
	}
	if got != "remote.png" {
		t.Errorf("got %q, want %q (base-name fallback)", got, "remote.png")
	}

	// Same with a relative ".." path.
	got2, err := cwdRelative("../something/else.png")
	if err != nil {
		t.Fatal(err)
	}
	if got2 != "else.png" {
		t.Errorf("got %q, want %q (base-name fallback for ../)", got2, "else.png")
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		// 45.2 KB matches the design example: 46285 / 1024 ≈ 45.20.
		{46285, "45.2 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024*1024 + 512*1024, "1.5 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.in), func(t *testing.T) {
			t.Parallel()
			if got := formatBytes(tt.in); got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func sampleMedias() []*zendesk.GuideMedia {
	return []*zendesk.GuideMedia{
		{ID: "01ABC", Name: "images/a.png", URL: "https://hc/a", Version: 1},
		{ID: "01DEF", Name: "images/b.png", URL: "https://hc/b", Version: 2},
	}
}

func TestPrintMediaListTable(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := printMediaListTable(&buf, sampleMedias()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"LOCAL PATH", "GUIDE MEDIA ID", "URL", "VERSION",
		"images/a.png", "01ABC", "https://hc/a",
		"images/b.png", "01DEF", "https://hc/b",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestPrintMediaListTable_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := printMediaListTable(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "LOCAL PATH") {
		t.Errorf("empty list should still print header, got %q", buf.String())
	}
}

func TestPrintMediaListYAML_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := printMediaListYAML(&buf, nil); err != nil {
		t.Fatal(err)
	}
	// Output must be valid YAML that round-trips to an empty map; we don't
	// pin the exact representation ("{}" vs blank) but we forbid garbage.
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("yaml output is not valid: %q -> %v", buf.String(), err)
	}
	if len(parsed) != 0 {
		t.Errorf("empty input produced non-empty map: %v", parsed)
	}
}

func TestPrintMediaListMarkdown_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := printMediaListMarkdown(&buf, nil); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + sep) for empty list, got %d:\n%s", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], "| LOCAL PATH ") {
		t.Errorf("header line = %q", lines[0])
	}
	if lines[1] != "|---|---|---|---|" {
		t.Errorf("separator line = %q", lines[1])
	}
}

func TestPrintMediaListYAML(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := printMediaListYAML(&buf, sampleMedias()); err != nil {
		t.Fatal(err)
	}

	// Round-trip through YAML to verify structure and types.
	var parsed map[string]struct {
		MediaID string `yaml:"media_id"`
		URL     string `yaml:"url"`
		Version int    `yaml:"version"`
	}
	if err := yaml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("yaml.Unmarshal(%q): %v", buf.String(), err)
	}
	if got := parsed["images/a.png"]; got.MediaID != "01ABC" || got.URL != "https://hc/a" || got.Version != 1 {
		t.Errorf("entry a = %+v", got)
	}
	if got := parsed["images/b.png"]; got.MediaID != "01DEF" || got.URL != "https://hc/b" || got.Version != 2 {
		t.Errorf("entry b = %+v", got)
	}
}

func TestPrintMediaListMarkdown(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := printMediaListMarkdown(&buf, sampleMedias()); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4 (header + sep + 2 rows): %q", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], "| LOCAL PATH ") {
		t.Errorf("header line = %q", lines[0])
	}
	if lines[1] != "|---|---|---|---|" {
		t.Errorf("separator line = %q", lines[1])
	}
	if !strings.Contains(lines[2], "01ABC") || !strings.Contains(lines[3], "01DEF") {
		t.Errorf("rows missing IDs:\n%s", buf.String())
	}
}

// ------------------- MediaCreateCmd.Run -------------------

func writeMediaFile(t *testing.T, name string, contents []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMediaCreateCmd_Run_DryRunDoesNotCallClient(t *testing.T) {
	path := writeMediaFile(t, "test.png", pngMagic)
	mock := &testhelper.MockZendeskClient{
		CreateUploadURLFunc: func(string, int64) (*zendesk.UploadURLResponse, error) {
			t.Fatal("CreateUploadURL must not be called in dry-run")
			return nil, nil
		},
		UploadFileBinaryFunc: func(string, map[string]string, io.Reader) error {
			t.Fatal("UploadFileBinary must not be called in dry-run")
			return nil
		},
		CreateGuideMediaFunc: func(string, string) (*zendesk.GuideMedia, error) {
			t.Fatal("CreateGuideMedia must not be called in dry-run")
			return nil, nil
		},
	}
	cmd := MediaCreateCmd{File: path, DryRun: true, client: mock}
	if err := cmd.Run(); err != nil {
		t.Fatalf("dry-run Run: %v", err)
	}
}

func TestMediaCreateCmd_Run_Success(t *testing.T) {
	path := writeMediaFile(t, "screenshot.png", pngMagic)

	var (
		callOrder        []string
		gotContentType   string
		gotFileSize      int64
		gotUploadURL     string
		gotHeaders       map[string]string
		gotUploadedBody  []byte
		gotAssetUploadID string
		gotFilename      string
	)
	mock := &testhelper.MockZendeskClient{
		CreateUploadURLFunc: func(ct string, sz int64) (*zendesk.UploadURLResponse, error) {
			callOrder = append(callOrder, "CreateUploadURL")
			gotContentType = ct
			gotFileSize = sz
			return &zendesk.UploadURLResponse{
				UploadURL:     "https://s3.example/upload?sig=abc",
				AssetUploadID: "01ASSET",
				Headers:       map[string]string{"Content-Type": "image/png", "X-Amz-Date": "20260101T000000Z"},
			}, nil
		},
		UploadFileBinaryFunc: func(url string, headers map[string]string, body io.Reader) error {
			callOrder = append(callOrder, "UploadFileBinary")
			gotUploadURL = url
			gotHeaders = headers
			gotUploadedBody, _ = io.ReadAll(body)
			return nil
		},
		CreateGuideMediaFunc: func(assetID, filename string) (*zendesk.GuideMedia, error) {
			callOrder = append(callOrder, "CreateGuideMedia")
			gotAssetUploadID = assetID
			gotFilename = filename
			return &zendesk.GuideMedia{ID: "01HM", URL: "https://hc/abc", Version: 1}, nil
		},
	}

	cmd := MediaCreateCmd{File: path, client: mock}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	wantOrder := []string{"CreateUploadURL", "UploadFileBinary", "CreateGuideMedia"}
	if !reflect.DeepEqual(callOrder, wantOrder) {
		t.Errorf("call order = %v, want %v", callOrder, wantOrder)
	}
	if gotContentType != "image/png" {
		t.Errorf("CreateUploadURL contentType = %q", gotContentType)
	}
	if gotFileSize != int64(len(pngMagic)) {
		t.Errorf("CreateUploadURL fileSize = %d, want %d", gotFileSize, len(pngMagic))
	}
	if gotUploadURL != "https://s3.example/upload?sig=abc" {
		t.Errorf("UploadFileBinary uploadURL = %q", gotUploadURL)
	}
	// Headers from upload_url must be forwarded verbatim.
	if gotHeaders["X-Amz-Date"] != "20260101T000000Z" {
		t.Errorf("X-Amz-Date header lost: %v", gotHeaders)
	}
	// Body sent to S3 must match the file contents.
	if !bytes.Equal(gotUploadedBody, pngMagic) {
		t.Errorf("uploaded body mismatch\n got: %q\nwant: %q", gotUploadedBody, pngMagic)
	}
	if gotAssetUploadID != "01ASSET" {
		t.Errorf("CreateGuideMedia assetUploadID = %q", gotAssetUploadID)
	}
	// File is in t.TempDir() outside the test's cwd, so cwdRelative falls back to base name.
	if gotFilename != "screenshot.png" {
		t.Errorf("CreateGuideMedia filename = %q, want %q", gotFilename, "screenshot.png")
	}
}

func TestMediaCreateCmd_Run_UnsupportedMIMEType(t *testing.T) {
	path := writeMediaFile(t, "doc.txt", []byte("plain text content not an image"))
	mock := &testhelper.MockZendeskClient{
		CreateUploadURLFunc: func(string, int64) (*zendesk.UploadURLResponse, error) {
			t.Fatal("client must not be called for unsupported MIME types")
			return nil, nil
		},
	}
	cmd := MediaCreateCmd{File: path, client: mock}
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for unsupported MIME type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported file type") {
		t.Errorf("err = %q, want phrase 'unsupported file type'", err.Error())
	}
}

func TestMediaCreateCmd_Run_RetryExhaustionMessage(t *testing.T) {
	path := writeMediaFile(t, "test.png", pngMagic)

	calls := 0
	mock := &testhelper.MockZendeskClient{
		CreateUploadURLFunc: func(string, int64) (*zendesk.UploadURLResponse, error) {
			calls++
			return nil, &zendesk.HTTPError{StatusCode: 503}
		},
	}
	cmd := MediaCreateCmd{File: path, client: mock}
	err := cmd.Run()

	if calls != mediaMaxRetries+1 {
		t.Errorf("CreateUploadURL calls = %d, want %d", calls, mediaMaxRetries+1)
	}
	if err == nil {
		t.Fatal("expected error from retry exhaustion, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create Guide Media") {
		t.Errorf("err = %q, want 'failed to create Guide Media'", err.Error())
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("gave up after %d retries", mediaMaxRetries)) {
		t.Errorf("err = %q, missing retry count phrase", err.Error())
	}
	if !strings.Contains(err.Error(), "Please retry the command") {
		t.Errorf("err = %q, missing retry hint", err.Error())
	}
}

// Critical regression test: when Step 2 (UploadFileBinary) fails transiently,
// the retry must restart from Step 1 AND rewind the file so the second upload
// sends the full file content again. Without f.Seek(0) before UploadFileBinary,
// the second iteration would send empty bytes (file at EOF after first read).
func TestMediaCreateCmd_Run_RetriesAndSucceeds(t *testing.T) {
	path := writeMediaFile(t, "test.png", pngMagic)

	var (
		createUploadURLCalls  int
		uploadFileBinaryCalls int
		createGuideMediaCalls int
		secondUploadBody      []byte
		assetIDsSeenByCreate  []string
	)
	mock := &testhelper.MockZendeskClient{
		CreateUploadURLFunc: func(ct string, sz int64) (*zendesk.UploadURLResponse, error) {
			createUploadURLCalls++
			// Each retry should produce a fresh asset_upload_id, mirroring real Zendesk behavior.
			return &zendesk.UploadURLResponse{
				UploadURL:     "https://upload.example/x",
				AssetUploadID: fmt.Sprintf("asset-%d", createUploadURLCalls),
			}, nil
		},
		UploadFileBinaryFunc: func(_ string, _ map[string]string, body io.Reader) error {
			uploadFileBinaryCalls++
			// Drain the body to advance the file's read position — simulates a real upload that
			// reads the whole file before failing.
			data, _ := io.ReadAll(body)
			if uploadFileBinaryCalls == 1 {
				return &zendesk.HTTPError{StatusCode: 503}
			}
			secondUploadBody = data
			return nil
		},
		CreateGuideMediaFunc: func(assetID, _ string) (*zendesk.GuideMedia, error) {
			createGuideMediaCalls++
			assetIDsSeenByCreate = append(assetIDsSeenByCreate, assetID)
			return &zendesk.GuideMedia{ID: "01HM", URL: "https://hc/x", Version: 1}, nil
		},
	}

	cmd := MediaCreateCmd{File: path, client: mock}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run failed unexpectedly: %v", err)
	}

	if createUploadURLCalls != 2 {
		t.Errorf("CreateUploadURL calls = %d, want 2 (retry must restart from Step 1)", createUploadURLCalls)
	}
	if uploadFileBinaryCalls != 2 {
		t.Errorf("UploadFileBinary calls = %d, want 2", uploadFileBinaryCalls)
	}
	if createGuideMediaCalls != 1 {
		t.Errorf("CreateGuideMedia calls = %d, want 1 (Step 3 must run only after a successful upload)", createGuideMediaCalls)
	}

	// The pivotal assertion: on retry, the file MUST be rewound. Without Seek(0),
	// the second UploadFileBinary call would receive an empty body.
	if !bytes.Equal(secondUploadBody, pngMagic) {
		t.Errorf("second upload body = %q (len=%d), want full file %q (len=%d) — file was not rewound between retries",
			secondUploadBody, len(secondUploadBody), pngMagic, len(pngMagic))
	}

	// And Step 3 must use the FRESH asset_upload_id from the retry, not the stale one.
	if len(assetIDsSeenByCreate) != 1 || assetIDsSeenByCreate[0] != "asset-2" {
		t.Errorf("CreateGuideMedia received assetIDs %v, want [asset-2] (must use the retried Step 1 result)", assetIDsSeenByCreate)
	}
}

func TestMediaCreateCmd_Run_NonRetryableErrorPropagatesAsIs(t *testing.T) {
	path := writeMediaFile(t, "test.png", pngMagic)

	calls := 0
	mock := &testhelper.MockZendeskClient{
		CreateUploadURLFunc: func(string, int64) (*zendesk.UploadURLResponse, error) {
			calls++
			return nil, &zendesk.HTTPError{StatusCode: 401}
		},
	}
	cmd := MediaCreateCmd{File: path, client: mock}
	err := cmd.Run()

	if calls != 1 {
		t.Errorf("calls = %d, want 1 (non-retryable must not retry)", calls)
	}
	var httpErr *zendesk.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != 401 {
		t.Errorf("expected 401 *HTTPError, got %v", err)
	}
}

// ------------------- MediaUpdateCmd.Run -------------------

func TestMediaUpdateCmd_Run_DryRunDoesNotCallClient(t *testing.T) {
	path := writeMediaFile(t, "test.png", pngMagic)
	mock := &testhelper.MockZendeskClient{
		CreateUploadURLFunc: func(string, int64) (*zendesk.UploadURLResponse, error) {
			t.Fatal("must not call client in dry-run")
			return nil, nil
		},
	}
	cmd := MediaUpdateCmd{File: path, MediaID: "01HM", DryRun: true, client: mock}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestMediaUpdateCmd_Run_CallsReplaceWithCorrectArgs(t *testing.T) {
	path := writeMediaFile(t, "v2.png", pngMagic)

	var (
		callOrder       []string
		gotMediaID      string
		gotAssetID      string
		gotFilename     string
		createCalled    bool
	)
	mock := &testhelper.MockZendeskClient{
		CreateUploadURLFunc: func(string, int64) (*zendesk.UploadURLResponse, error) {
			callOrder = append(callOrder, "CreateUploadURL")
			return &zendesk.UploadURLResponse{
				UploadURL:     "https://s3/x",
				AssetUploadID: "01NEWASSET",
			}, nil
		},
		UploadFileBinaryFunc: func(string, map[string]string, io.Reader) error {
			callOrder = append(callOrder, "UploadFileBinary")
			return nil
		},
		ReplaceGuideMediaFunc: func(id, assetID, filename string) (*zendesk.GuideMedia, error) {
			callOrder = append(callOrder, "ReplaceGuideMedia")
			gotMediaID = id
			gotAssetID = assetID
			gotFilename = filename
			return &zendesk.GuideMedia{ID: id, URL: "https://hc/x", Version: 2}, nil
		},
		CreateGuideMediaFunc: func(string, string) (*zendesk.GuideMedia, error) {
			createCalled = true
			return nil, nil
		},
	}

	cmd := MediaUpdateCmd{File: path, MediaID: "01TARGET", client: mock}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if createCalled {
		t.Errorf("update must call ReplaceGuideMedia, not CreateGuideMedia")
	}
	wantOrder := []string{"CreateUploadURL", "UploadFileBinary", "ReplaceGuideMedia"}
	if !reflect.DeepEqual(callOrder, wantOrder) {
		t.Errorf("call order = %v, want %v", callOrder, wantOrder)
	}
	if gotMediaID != "01TARGET" {
		t.Errorf("ReplaceGuideMedia id = %q, want %q", gotMediaID, "01TARGET")
	}
	if gotAssetID != "01NEWASSET" {
		t.Errorf("ReplaceGuideMedia assetID = %q", gotAssetID)
	}
	if gotFilename != "v2.png" {
		t.Errorf("ReplaceGuideMedia filename = %q, want %q", gotFilename, "v2.png")
	}
}

func TestMediaUpdateCmd_Run_RetryExhaustionMessage(t *testing.T) {
	path := writeMediaFile(t, "test.png", pngMagic)
	mock := &testhelper.MockZendeskClient{
		CreateUploadURLFunc: func(string, int64) (*zendesk.UploadURLResponse, error) {
			return nil, &zendesk.HTTPError{StatusCode: 503}
		},
	}
	cmd := MediaUpdateCmd{File: path, MediaID: "01HM", client: mock}
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to replace Guide Media") {
		t.Errorf("err = %q, want 'failed to replace Guide Media' (not 'create')", err.Error())
	}
}

// ------------------- MediaListCmd.Run -------------------

func TestMediaListCmd_Run_TableFormat(t *testing.T) {
	t.Parallel()
	mock := &testhelper.MockZendeskClient{
		ListGuideMediasFunc: func() ([]*zendesk.GuideMedia, error) {
			return sampleMedias(), nil
		},
	}
	cmd := MediaListCmd{Output: "table", client: mock}
	// Output goes to stdout; we already test the formatter directly. Here we
	// just ensure Run does not error when the API returns rows.
	if err := cmd.Run(); err != nil {
		t.Errorf("Run: %v", err)
	}
}

func TestMediaListCmd_Run_PropagatesAPIError(t *testing.T) {
	t.Parallel()
	want := errors.New("api unreachable")
	mock := &testhelper.MockZendeskClient{
		ListGuideMediasFunc: func() ([]*zendesk.GuideMedia, error) {
			return nil, want
		},
	}
	cmd := MediaListCmd{Output: "table", client: mock}
	err := cmd.Run()
	if !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
}

func TestMediaListCmd_Run_AcceptsAllFormats(t *testing.T) {
	mock := &testhelper.MockZendeskClient{
		ListGuideMediasFunc: func() ([]*zendesk.GuideMedia, error) {
			return sampleMedias(), nil
		},
	}
	for _, format := range []string{"table", "yaml", "markdown", ""} {
		cmd := MediaListCmd{Output: format, client: mock}
		if err := cmd.Run(); err != nil {
			t.Errorf("format %q: %v", format, err)
		}
	}
}

// ------------------- MediaDeleteCmd.Run -------------------

func TestMediaDeleteCmd_Run_DryRunDoesNotCallClient(t *testing.T) {
	t.Parallel()
	mock := &testhelper.MockZendeskClient{
		DeleteGuideMediaFunc: func(string) error {
			t.Fatal("must not call delete in dry-run")
			return nil
		},
	}
	cmd := MediaDeleteCmd{MediaID: "01HM", DryRun: true, client: mock}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestMediaDeleteCmd_Run_Success(t *testing.T) {
	t.Parallel()
	var gotID string
	mock := &testhelper.MockZendeskClient{
		DeleteGuideMediaFunc: func(id string) error {
			gotID = id
			return nil
		},
	}
	cmd := MediaDeleteCmd{MediaID: "01HM", client: mock}
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gotID != "01HM" {
		t.Errorf("DeleteGuideMedia id = %q", gotID)
	}
}

func TestMediaDeleteCmd_Run_PropagatesError(t *testing.T) {
	t.Parallel()
	want := &zendesk.HTTPError{StatusCode: 404}
	mock := &testhelper.MockZendeskClient{
		DeleteGuideMediaFunc: func(string) error { return want },
	}
	cmd := MediaDeleteCmd{MediaID: "01HM", client: mock}
	err := cmd.Run()
	var httpErr *zendesk.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != 404 {
		t.Errorf("err = %v, want 404 *HTTPError", err)
	}
}
