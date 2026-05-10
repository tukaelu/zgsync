package cli

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/tukaelu/zgsync/internal/zendesk"
	"gopkg.in/yaml.v3"
)

const mediaMaxRetries = 3

var allowedMediaMIMETypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

type CommandMedia struct {
	Create MediaCreateCmd `cmd:"" help:"Upload an image and create a Guide Media."`
	Update MediaUpdateCmd `cmd:"" help:"Replace an existing Guide Media."`
	List   MediaListCmd   `cmd:"" help:"List uploaded Guide Media as a mapping."`
	Delete MediaDeleteCmd `cmd:"" help:"Delete a Guide Media."`
}

type MediaCreateCmd struct {
	DryRun bool           `name:"dry-run" help:"Dry run (no upload)."`
	File   string         `arg:"" help:"Image file to upload." type:"existingfile"`
	client zendesk.Client `kong:"-"`
}

func (c *MediaCreateCmd) AfterApply(g *Global) error {
	c.client = zendesk.NewClient(g.Config.Subdomain, g.Config.Email, g.Config.Token)
	return nil
}

func (c *MediaCreateCmd) Run() error {
	f, info, err := openMediaFile(c.File)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	mimeType, err := detectMediaMIMEType(f)
	if err != nil {
		return err
	}
	if !allowedMediaMIMETypes[mimeType] {
		return fmt.Errorf("unsupported file type: %s (allowed: image/png, image/jpeg, image/gif, image/webp)", mimeType)
	}

	nameForUpload, err := cwdRelative(c.File)
	if err != nil {
		return err
	}

	if c.DryRun {
		fmt.Printf("[dry-run] Would upload %s (%s) as %q\n", c.File, formatBytes(info.Size()), nameForUpload)
		return nil
	}

	fmt.Printf("Uploading %s...\n", c.File)

	var media *zendesk.GuideMedia
	err = withMediaRetry(mediaMaxRetries, func() error {
		upload, err := c.client.CreateUploadURL(mimeType, info.Size())
		if err != nil {
			return err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := c.client.UploadFileBinary(upload.UploadURL, upload.Headers, f); err != nil {
			return err
		}
		m, err := c.client.CreateGuideMedia(upload.AssetUploadID, nameForUpload)
		if err != nil {
			return err
		}
		media = m
		return nil
	})
	if err != nil {
		var exhausted *errRetriesExhausted
		if errors.As(err, &exhausted) {
			return fmt.Errorf("failed to create Guide Media (gave up after %d retries).\nPlease retry the command", mediaMaxRetries)
		}
		return err
	}

	fmt.Printf("  Guide Media ID:  %s\n", media.ID)
	fmt.Printf("  URL:             %s\n", media.URL)
	return nil
}

type MediaUpdateCmd struct {
	MediaID string         `name:"media-id" required:"" help:"Guide Media ID to replace (ULID)."`
	DryRun  bool           `name:"dry-run" help:"Dry run (no upload)."`
	File    string         `arg:"" help:"Image file to upload." type:"existingfile"`
	client  zendesk.Client `kong:"-"`
}

func (c *MediaUpdateCmd) AfterApply(g *Global) error {
	c.client = zendesk.NewClient(g.Config.Subdomain, g.Config.Email, g.Config.Token)
	return nil
}

func (c *MediaUpdateCmd) Run() error {
	f, info, err := openMediaFile(c.File)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	mimeType, err := detectMediaMIMEType(f)
	if err != nil {
		return err
	}
	if !allowedMediaMIMETypes[mimeType] {
		return fmt.Errorf("unsupported file type: %s (allowed: image/png, image/jpeg, image/gif, image/webp)", mimeType)
	}

	nameForUpload, err := cwdRelative(c.File)
	if err != nil {
		return err
	}

	if c.DryRun {
		fmt.Printf("[dry-run] Would replace Guide Media %s with %s (%s)\n", c.MediaID, c.File, formatBytes(info.Size()))
		return nil
	}

	fmt.Printf("Uploading %s...\n", c.File)

	var media *zendesk.GuideMedia
	err = withMediaRetry(mediaMaxRetries, func() error {
		upload, err := c.client.CreateUploadURL(mimeType, info.Size())
		if err != nil {
			return err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := c.client.UploadFileBinary(upload.UploadURL, upload.Headers, f); err != nil {
			return err
		}
		m, err := c.client.ReplaceGuideMedia(c.MediaID, upload.AssetUploadID, nameForUpload)
		if err != nil {
			return err
		}
		media = m
		return nil
	})
	if err != nil {
		var exhausted *errRetriesExhausted
		if errors.As(err, &exhausted) {
			return fmt.Errorf("failed to replace Guide Media (gave up after %d retries).\nPlease retry the command", mediaMaxRetries)
		}
		return err
	}

	fmt.Printf("  Guide Media ID:  %s\n", media.ID)
	fmt.Printf("  URL:             %s\n", media.URL)
	fmt.Printf("  Version:         %d\n", media.Version)
	return nil
}

type MediaListCmd struct {
	Output string         `name:"output" default:"table" enum:"table,yaml,markdown" help:"Output format (table, yaml, or markdown)."`
	client zendesk.Client `kong:"-"`
}

func (c *MediaListCmd) AfterApply(g *Global) error {
	c.client = zendesk.NewClient(g.Config.Subdomain, g.Config.Email, g.Config.Token)
	return nil
}

func (c *MediaListCmd) Run() error {
	medias, err := c.client.ListGuideMedias()
	if err != nil {
		return err
	}
	sort.Slice(medias, func(i, j int) bool { return medias[i].Name < medias[j].Name })

	switch c.Output {
	case "yaml":
		return printMediaListYAML(os.Stdout, medias)
	case "markdown":
		return printMediaListMarkdown(os.Stdout, medias)
	default:
		return printMediaListTable(os.Stdout, medias)
	}
}

type MediaDeleteCmd struct {
	MediaID string         `name:"media-id" required:"" help:"Guide Media ID to delete (ULID)."`
	DryRun  bool           `name:"dry-run" help:"Dry run (no deletion)."`
	client  zendesk.Client `kong:"-"`
}

func (c *MediaDeleteCmd) AfterApply(g *Global) error {
	c.client = zendesk.NewClient(g.Config.Subdomain, g.Config.Email, g.Config.Token)
	return nil
}

func (c *MediaDeleteCmd) Run() error {
	if c.DryRun {
		fmt.Printf("[dry-run] Would delete Guide Media %s\n", c.MediaID)
		return nil
	}
	fmt.Println("Warning: Deleting a Guide Media that is embedded in articles will break those images.")
	fmt.Println("Deleting...")
	fmt.Printf("  Guide Media ID:  %s\n", c.MediaID)
	if err := c.client.DeleteGuideMedia(c.MediaID); err != nil {
		return err
	}
	fmt.Println("  Deleted.")
	return nil
}

// errRetriesExhausted indicates that withMediaRetry gave up after exhausting
// the retry budget. The wrapped error is the last failure encountered.
type errRetriesExhausted struct {
	retries int
	lastErr error
}

func (e *errRetriesExhausted) Error() string {
	return fmt.Sprintf("gave up after %d retries: %v", e.retries, e.lastErr)
}

func (e *errRetriesExhausted) Unwrap() error { return e.lastErr }

// withMediaRetry runs fn, retrying on retryable errors up to maxRetries times.
// Total invocations are at most maxRetries+1. Non-retryable errors return immediately.
func withMediaRetry(maxRetries int, fn func() error) error {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if lastErr = fn(); lastErr == nil {
			return nil
		}
		if !isRetryable(lastErr) {
			return lastErr
		}
	}
	return &errRetriesExhausted{retries: maxRetries, lastErr: lastErr}
}

func isRetryable(err error) bool {
	var httpErr *zendesk.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= 500
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

func openMediaFile(path string) (*os.File, os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	return f, info, nil
}

// detectMediaMIMEType uses http.DetectContentType on the first 512 bytes,
// falling back to the file extension when detection fails.
// The file is rewound to the start before returning.
func detectMediaMIMEType(f *os.File) (string, error) {
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	mimeType := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(mimeType, "image/") {
		switch strings.ToLower(filepath.Ext(f.Name())) {
		case ".png":
			return "image/png", nil
		case ".jpg", ".jpeg":
			return "image/jpeg", nil
		case ".gif":
			return "image/gif", nil
		case ".webp":
			return "image/webp", nil
		}
	}
	return mimeType, nil
}

// cwdRelative returns the path relative to the current working directory.
// If the file lives outside the CWD subtree (e.g. "../foo" or an absolute
// path elsewhere), it falls back to the base name to keep the Guide Media
// name field tidy.
func cwdRelative(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(cwd, abs)
	if err != nil {
		return "", err
	}
	if !filepath.IsLocal(rel) {
		return filepath.Base(path), nil
	}
	return rel, nil
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func printMediaListTable(w io.Writer, medias []*zendesk.GuideMedia) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "LOCAL PATH\tGUIDE MEDIA ID\tURL\tVERSION"); err != nil {
		return err
	}
	for _, m := range medias {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%d\n", m.Name, m.ID, m.URL, m.Version); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func printMediaListYAML(w io.Writer, medias []*zendesk.GuideMedia) error {
	root := &yaml.Node{Kind: yaml.MappingNode}
	for _, m := range medias {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: m.Name},
			&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "media_id"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: m.ID},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "url"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: m.URL},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "version"},
				{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(m.Version)},
			}},
		)
	}
	out, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

func printMediaListMarkdown(w io.Writer, medias []*zendesk.GuideMedia) error {
	if _, err := fmt.Fprintln(w, "| LOCAL PATH | GUIDE MEDIA ID | URL | VERSION |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|---|---|---|---|"); err != nil {
		return err
	}
	for _, m := range medias {
		if _, err := fmt.Fprintf(w, "| %s | %s | %s | %d |\n", m.Name, m.ID, m.URL, m.Version); err != nil {
			return err
		}
	}
	return nil
}
