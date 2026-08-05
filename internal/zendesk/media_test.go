package zendesk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// readJSONBody decodes the request body into v. Fails the test if parsing errors.
func readJSONBody(t *testing.T, r *http.Request, v interface{}) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("unmarshal body %q: %v", string(body), err)
	}
}

func TestClient_CreateUploadURL(t *testing.T) {
	t.Parallel()

	var (
		gotMethod, gotPath string
		gotBody            uploadURLReqBody
		rawBody            []byte
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		rawBody, _ = io.ReadAll(r.Body)
		_ = json.Unmarshal(rawBody, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		// Real response shape: wrapped under "upload_url"; "headers" is a
		// JSON-encoded string, not an object.
		_, _ = w.Write([]byte(`{
			"upload_url": {
				"url": "https://upload-service.zendesk.com/abc?sig=xyz",
				"asset_upload_id": "01HASSET",
				"headers": "{\"Content-Type\":\"image/png\",\"X-Amz-Date\":\"20260101T000000Z\",\"X-Amz-Server-Side-Encryption\":\"AES256\"}"
			}
		}`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	resp, err := c.CreateUploadURL("image/png", 12345)
	if err != nil {
		t.Fatalf("CreateUploadURL: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v2/guide/medias/upload_url" {
		t.Errorf("path = %q", gotPath)
	}
	// Request body must be FLAT — Go's json.Unmarshal silently ignores extra
	// wrapper keys, so verify against the raw bytes too.
	if gotBody.ContentType != "image/png" || gotBody.FileSize != 12345 {
		t.Errorf("body = %+v", gotBody)
	}
	if strings.Contains(string(rawBody), `"upload_url"`) {
		t.Errorf("request body unexpectedly wrapped under upload_url: %s", rawBody)
	}
	if resp.UploadURL != "https://upload-service.zendesk.com/abc?sig=xyz" {
		t.Errorf("UploadURL = %q", resp.UploadURL)
	}
	if resp.AssetUploadID != "01HASSET" {
		t.Errorf("AssetUploadID = %q", resp.AssetUploadID)
	}
	for k, want := range map[string]string{
		"Content-Type":                 "image/png",
		"X-Amz-Date":                   "20260101T000000Z",
		"X-Amz-Server-Side-Encryption": "AES256",
	} {
		if got := resp.Headers[k]; got != want {
			t.Errorf("Headers[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestClient_CreateUploadURL_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	_, err := c.CreateUploadURL("image/png", 100)

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *HTTPError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d", httpErr.StatusCode)
	}
}

func TestClient_CreateUploadURL_MalformedOuterJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	if _, err := c.CreateUploadURL("image/png", 100); err == nil {
		t.Fatal("expected error from malformed outer JSON, got nil")
	}
}

// The inner headers field is itself a JSON-encoded string. If it cannot be
// parsed as a header map, CreateUploadURL must surface an error rather than
// silently dropping the signed headers (which would break the S3 upload).
func TestClient_CreateUploadURL_MalformedHeadersString(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"upload_url": {
				"url": "https://upload-service.zendesk.com/x",
				"asset_upload_id": "01HASSET",
				"headers": "not-a-json-object"
			}
		}`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	_, err := c.CreateUploadURL("image/png", 100)
	if err == nil {
		t.Fatal("expected error from malformed headers string, got nil")
	}
	if !strings.Contains(err.Error(), "headers") {
		t.Errorf("err = %q, expected mention of 'headers'", err.Error())
	}
}

// Empty headers string should produce a nil/empty Headers map rather than
// erroring — defensive against minor doc/spec drift.
func TestClient_CreateUploadURL_EmptyHeadersString(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"upload_url": {
				"url": "https://upload-service.zendesk.com/x",
				"asset_upload_id": "01HASSET",
				"headers": ""
			}
		}`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	resp, err := c.CreateUploadURL("image/png", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UploadURL == "" {
		t.Errorf("UploadURL should still be parsed even with empty headers")
	}
	if len(resp.Headers) != 0 {
		t.Errorf("Headers should be empty, got %v", resp.Headers)
	}
}

func TestClient_UploadFileBinary(t *testing.T) {
	t.Parallel()

	var (
		gotMethod        string
		gotHeaders       http.Header
		gotBody          []byte
		gotContentLength int64
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotHeaders = r.Header.Clone()
		gotContentLength = r.ContentLength
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClientImpl("")
	payload := []byte("PNG-binary-data-here")
	headers := map[string]string{
		"Content-Type":                 "image/png",
		"X-Amz-Date":                   "20260101T000000Z",
		"X-Amz-Server-Side-Encryption": "AES256",
	}
	if err := c.UploadFileBinary(server.URL, headers, bytes.NewReader(payload)); err != nil {
		t.Fatalf("UploadFileBinary: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if !bytes.Equal(gotBody, payload) {
		t.Errorf("body mismatch: got %q", gotBody)
	}
	if gotContentLength != int64(len(payload)) {
		t.Errorf("Content-Length = %d, want %d", gotContentLength, len(payload))
	}
	for k, want := range headers {
		if got := gotHeaders.Get(k); got != want {
			t.Errorf("header %q = %q, want %q", k, got, want)
		}
	}
}

func TestClient_UploadFileBinary_S3Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := newTestClientImpl("")
	err := c.UploadFileBinary(server.URL, nil, bytes.NewReader([]byte("x")))

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *HTTPError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d", httpErr.StatusCode)
	}
}

// nonSeekerReader wraps a Reader to deliberately hide ReadSeeker so we can
// verify that Content-Length is omitted when the body is not seekable.
type nonSeekerReader struct{ r io.Reader }

func (n *nonSeekerReader) Read(p []byte) (int, error) { return n.r.Read(p) }

func TestClient_UploadFileBinary_NonSeekableNoContentLength(t *testing.T) {
	t.Parallel()

	var gotContentLength int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentLength = r.ContentLength
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClientImpl("")
	payload := []byte("not seekable")
	body := &nonSeekerReader{r: bytes.NewReader(payload)}
	if err := c.UploadFileBinary(server.URL, nil, body); err != nil {
		t.Fatalf("UploadFileBinary: %v", err)
	}

	if gotContentLength == int64(len(payload)) {
		t.Errorf("Content-Length unexpectedly set to %d for non-seekable body", gotContentLength)
	}
}

func TestClient_CreateGuideMedia(t *testing.T) {
	t.Parallel()

	var (
		gotMethod, gotPath string
		gotBody            guideMediaReqBody
		// Capture raw body to assert it's NOT wrapped under "media" or "guide_media".
		rawBody []byte
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		rawBody, _ = io.ReadAll(r.Body)
		_ = json.Unmarshal(rawBody, &gotBody)
		// Real response wrapper key is "media" (not "guide_media").
		_, _ = w.Write([]byte(`{
			"media": {
				"access_key": "01ASSET",
				"content_type": "image/png",
				"created_at": "2026-05-10T00:00:00.000Z",
				"id": "01HMEDIA",
				"name": "uploads/foo.png",
				"size": 10394,
				"updated_at": "2026-05-10T00:00:00.000Z",
				"url": "/guide-media/01ASSET",
				"version": 1
			}
		}`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	media, err := c.CreateGuideMedia("01ASSET", "uploads/foo.png")
	if err != nil {
		t.Fatalf("CreateGuideMedia: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v2/guide/medias" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody.AssetUploadID != "01ASSET" || gotBody.Filename != "uploads/foo.png" {
		t.Errorf("body = %+v", gotBody)
	}
	// Request must NOT be wrapped under any key.
	for _, banned := range []string{`"media"`, `"guide_media"`} {
		if strings.Contains(string(rawBody), banned) {
			t.Errorf("request body unexpectedly wrapped (contains %s): %s", banned, rawBody)
		}
	}
	// Response must be unwrapped from "media".
	if media.ID != "01HMEDIA" || media.Name != "uploads/foo.png" || media.Version != 1 {
		t.Errorf("media = %+v", media)
	}
	if media.URL != "/guide-media/01ASSET" {
		t.Errorf("media.URL = %q", media.URL)
	}
}

func TestClient_CreateGuideMedia_Conflict(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	_, err := c.CreateGuideMedia("asset", "foo.png")

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 *HTTPError, got %v", err)
	}
}

func TestClient_ReplaceGuideMedia(t *testing.T) {
	t.Parallel()

	var (
		gotMethod, gotPath string
		gotBody            guideMediaReqBody
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		readJSONBody(t, r, &gotBody)
		_, _ = w.Write([]byte(`{
			"media": {
				"id": "01HMEDIA",
				"name": "uploads/foo.png",
				"url": "/guide-media/01ASSET2",
				"version": 2
			}
		}`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	media, err := c.ReplaceGuideMedia("01HMEDIA", "01ASSET2", "uploads/foo.png")
	if err != nil {
		t.Fatalf("ReplaceGuideMedia: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/api/v2/guide/medias/01HMEDIA" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody.AssetUploadID != "01ASSET2" {
		t.Errorf("AssetUploadID = %q", gotBody.AssetUploadID)
	}
	if media.Version != 2 {
		t.Errorf("Version = %d, want 2", media.Version)
	}
}

func TestClient_DeleteGuideMedia(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		serverStatus int
		wantErrCode  int // 0 means expect no error
	}{
		{"success 204", http.StatusNoContent, 0},
		{"not found 404", http.StatusNotFound, http.StatusNotFound},
		{"server error 500", http.StatusInternalServerError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotMethod, gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			c := newTestClientImpl(server.URL)
			err := c.DeleteGuideMedia("01HMEDIA")

			if tt.wantErrCode == 0 {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
			} else {
				var httpErr *HTTPError
				if !errors.As(err, &httpErr) || httpErr.StatusCode != tt.wantErrCode {
					t.Errorf("expected %d *HTTPError, got %v", tt.wantErrCode, err)
				}
			}
			if gotMethod != http.MethodDelete {
				t.Errorf("method = %q, want DELETE", gotMethod)
			}
			if gotPath != "/api/v2/guide/medias/01HMEDIA" {
				t.Errorf("path = %q", gotPath)
			}
		})
	}
}

func TestClient_ListGuideMedias_SinglePage(t *testing.T) {
	t.Parallel()

	var requestCount int
	var firstQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount == 0 {
			firstQuery = r.URL.RawQuery
		}
		requestCount++
		// Real response wrapper is "records", not "guide_medias".
		_, _ = w.Write([]byte(`{
			"meta": {"has_more": false, "after_cursor": "", "before_cursor": ""},
			"records": [
				{"id":"01A","name":"a.png","url":"/guide-media/a","version":1},
				{"id":"01B","name":"b.png","url":"/guide-media/b","version":2}
			]
		}`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	medias, err := c.ListGuideMedias()
	if err != nil {
		t.Fatalf("ListGuideMedias: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("requestCount = %d, want 1", requestCount)
	}
	// page[size]=100 must be on the first request — sometimes percent-encoded.
	if !strings.Contains(firstQuery, "page%5Bsize%5D=100") && !strings.Contains(firstQuery, "page[size]=100") {
		t.Errorf("first query = %q, expected to include page[size]=100", firstQuery)
	}
	if len(medias) != 2 {
		t.Fatalf("len(medias) = %d, want 2", len(medias))
	}
	if medias[0].ID != "01A" || medias[1].ID != "01B" {
		t.Errorf("ids = %q,%q", medias[0].ID, medias[1].ID)
	}
}

func TestClient_ListGuideMedias_MultiPage(t *testing.T) {
	t.Parallel()

	var receivedCursors []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the cursor sent on each request so we can assert pagination.
		receivedCursors = append(receivedCursors, r.URL.Query().Get("page[after]"))
		switch r.URL.Query().Get("page[after]") {
		case "":
			_, _ = fmt.Fprint(w, `{
				"meta": {"has_more": true, "after_cursor": "MQ"},
				"records": [{"id":"01A","name":"a.png","url":"/guide-media/a","version":1}]
			}`)
		case "MQ":
			_, _ = fmt.Fprint(w, `{
				"meta": {"has_more": true, "after_cursor": "Mg"},
				"records": [{"id":"01B","name":"b.png","url":"/guide-media/b","version":1}]
			}`)
		case "Mg":
			_, _ = fmt.Fprint(w, `{
				"meta": {"has_more": false, "after_cursor": ""},
				"records": [{"id":"01C","name":"c.png","url":"/guide-media/c","version":1}]
			}`)
		default:
			t.Errorf("unexpected page[after] = %q", r.URL.Query().Get("page[after]"))
		}
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	medias, err := c.ListGuideMedias()
	if err != nil {
		t.Fatalf("ListGuideMedias: %v", err)
	}

	if got, want := len(receivedCursors), 3; got != want {
		t.Fatalf("requests = %d, want %d (cursors: %v)", got, want, receivedCursors)
	}
	if receivedCursors[0] != "" || receivedCursors[1] != "MQ" || receivedCursors[2] != "Mg" {
		t.Errorf("cursors received = %v, want [empty MQ Mg]", receivedCursors)
	}
	if len(medias) != 3 {
		t.Fatalf("len(medias) = %d, want 3", len(medias))
	}
	wantIDs := []string{"01A", "01B", "01C"}
	for i, want := range wantIDs {
		if medias[i].ID != want {
			t.Errorf("medias[%d].ID = %q, want %q", i, medias[i].ID, want)
		}
	}
}

// Cursors may include characters that need URL-encoding (e.g. '+', '/'). Verify
// that a cursor containing a '+' is correctly forwarded by the next request.
func TestClient_ListGuideMedias_URLEncodesCursor(t *testing.T) {
	t.Parallel()

	const trickyCursor = "abc+def/ghi="
	var receivedCursors []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCursors = append(receivedCursors, r.URL.Query().Get("page[after]"))
		if len(receivedCursors) == 1 {
			_, _ = fmt.Fprintf(w, `{"meta":{"has_more":true,"after_cursor":%q},"records":[]}`, trickyCursor)
			return
		}
		_, _ = fmt.Fprint(w, `{"meta":{"has_more":false,"after_cursor":""},"records":[]}`)
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	if _, err := c.ListGuideMedias(); err != nil {
		t.Fatal(err)
	}
	// The Go server side decodes query params, so r.URL.Query().Get reveals the
	// post-decode cursor. If we URL-encode correctly, this round-trips.
	if receivedCursors[1] != trickyCursor {
		t.Errorf("decoded cursor = %q, want %q (url encoding lost roundtrip)", receivedCursors[1], trickyCursor)
	}
	// Sanity check our own URL encoding logic too.
	if got := url.QueryEscape(trickyCursor); !strings.Contains(got, "%2B") {
		t.Errorf("url.QueryEscape did not encode '+': %q", got)
	}
}

// has_more=true but after_cursor="" must terminate pagination instead of
// looping forever (defensive against malformed server responses).
func TestClient_ListGuideMedias_StopsOnEmptyCursor(t *testing.T) {
	t.Parallel()

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount > 1 {
			t.Errorf("server called more than once: %d", requestCount)
		}
		_, _ = w.Write([]byte(`{
			"meta": {"has_more": true, "after_cursor": ""},
			"records": [{"id":"01A","name":"a.png","url":"/guide-media/a","version":1}]
		}`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	medias, err := c.ListGuideMedias()
	if err != nil {
		t.Fatalf("ListGuideMedias: %v", err)
	}
	if len(medias) != 1 {
		t.Errorf("len(medias) = %d, want 1", len(medias))
	}
}

func TestClient_ListGuideMedias_PropagatesError(t *testing.T) {
	t.Parallel()

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{
				"meta": {"has_more": true, "after_cursor": "MQ"},
				"records": [{"id":"01A","name":"a.png","url":"/guide-media/a","version":1}]
			}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	_, err := c.ListGuideMedias()

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 *HTTPError after page 2 failure, got %v", err)
	}
}

func TestClient_ListGuideMedias_MalformedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	if _, err := c.ListGuideMedias(); err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestClient_CreateGuideMedia_MalformedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	if _, err := c.CreateGuideMedia("asset", "foo.png"); err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestClient_ReplaceGuideMedia_MalformedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	if _, err := c.ReplaceGuideMedia("01HM", "asset", "foo.png"); err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestClient_ReplaceGuideMedia_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClientImpl(server.URL)
	_, err := c.ReplaceGuideMedia("01HM", "asset", "foo.png")

	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 *HTTPError, got %v", err)
	}
}

func TestClient_UploadFileBinary_TransportError(t *testing.T) {
	t.Parallel()

	// Start and immediately close the server so Do() fails on connect.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	c := newTestClientImpl("")
	err := c.UploadFileBinary(server.URL, nil, bytes.NewReader([]byte("x")))
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
}
