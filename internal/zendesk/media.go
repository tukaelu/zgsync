package zendesk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// UploadURLResponse holds the parsed response from
// POST /api/v2/guide/medias/upload_url. The Zendesk wire format wraps these
// fields under an "upload_url" object and serializes Headers as a JSON-encoded
// string; CreateUploadURL flattens that into this straightforward struct.
type UploadURLResponse struct {
	UploadURL     string
	AssetUploadID string
	Headers       map[string]string
}

// GuideMedia represents a Zendesk Guide Media object.
type GuideMedia struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	Version int    `json:"version"`
}

type uploadURLReqBody struct {
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
}

// uploadURLWireResp matches Zendesk's actual response shape:
//
//	{"upload_url":{"url":"...","asset_upload_id":"...","headers":"<json-encoded string>"}}
type uploadURLWireResp struct {
	UploadURL struct {
		URL           string `json:"url"`
		AssetUploadID string `json:"asset_upload_id"`
		Headers       string `json:"headers"`
	} `json:"upload_url"`
}

type guideMediaReqBody struct {
	AssetUploadID string `json:"asset_upload_id"`
	Filename      string `json:"filename"`
}

type wrappedGuideMediaResp struct {
	GuideMedia GuideMedia `json:"media"`
}

type guideMediaListResp struct {
	Records []GuideMedia `json:"records"`
	Meta    struct {
		HasMore     bool   `json:"has_more"`
		AfterCursor string `json:"after_cursor"`
	} `json:"meta"`
}

const guideMediaListPageSize = 100

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/guide_medias/#create-upload-url-for-a-guide-media-object
func (c *clientImpl) CreateUploadURL(contentType string, fileSize int64) (*UploadURLResponse, error) {
	b, err := json.Marshal(uploadURLReqBody{ContentType: contentType, FileSize: fileSize})
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequest(http.MethodPost, "/api/v2/guide/medias/upload_url", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	var wire uploadURLWireResp
	if err := json.Unmarshal([]byte(resp), &wire); err != nil {
		return nil, err
	}
	// The headers field is a JSON-encoded string, so it needs a second decode.
	var headers map[string]string
	if wire.UploadURL.Headers != "" {
		if err := json.Unmarshal([]byte(wire.UploadURL.Headers), &headers); err != nil {
			return nil, fmt.Errorf("decode upload_url headers: %w", err)
		}
	}
	return &UploadURLResponse{
		UploadURL:     wire.UploadURL.URL,
		AssetUploadID: wire.UploadURL.AssetUploadID,
		Headers:       headers,
	}, nil
}

// UploadFileBinary PUTs the body to the given signed upload URL with the
// supplied headers. If body implements io.ReadSeeker, Content-Length is set
// automatically.
func (c *clientImpl) UploadFileBinary(uploadURL string, headers map[string]string, body io.Reader) error {
	req, err := http.NewRequest(http.MethodPut, uploadURL, body)
	if err != nil {
		return err
	}
	if rs, ok := body.(io.ReadSeeker); ok {
		pos, _ := rs.Seek(0, io.SeekCurrent)
		end, _ := rs.Seek(0, io.SeekEnd)
		_, _ = rs.Seek(pos, io.SeekStart)
		req.ContentLength = end - pos
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return &HTTPError{StatusCode: res.StatusCode}
	}
	return nil
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/guide_medias/#create-guide-media
func (c *clientImpl) CreateGuideMedia(assetUploadID, filename string) (*GuideMedia, error) {
	b, err := json.Marshal(guideMediaReqBody{
		AssetUploadID: assetUploadID,
		Filename:      filename,
	})
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequest(http.MethodPost, "/api/v2/guide/medias", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	var wrapped wrappedGuideMediaResp
	if err := json.Unmarshal([]byte(resp), &wrapped); err != nil {
		return nil, err
	}
	return &wrapped.GuideMedia, nil
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/guide_medias/#replace-guide-media-object
func (c *clientImpl) ReplaceGuideMedia(id, assetUploadID, filename string) (*GuideMedia, error) {
	b, err := json.Marshal(guideMediaReqBody{
		AssetUploadID: assetUploadID,
		Filename:      filename,
	})
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequest(http.MethodPut, fmt.Sprintf("/api/v2/guide/medias/%s", id), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	var wrapped wrappedGuideMediaResp
	if err := json.Unmarshal([]byte(resp), &wrapped); err != nil {
		return nil, err
	}
	return &wrapped.GuideMedia, nil
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/guide_medias/#delete-guide-media-object
func (c *clientImpl) DeleteGuideMedia(id string) error {
	return c.doDeleteRequest(fmt.Sprintf("/api/v2/guide/medias/%s", id))
}

// refs: https://developer.zendesk.com/api-reference/help_center/help-center-api/guide_medias/#search-guide-media
// Pages through all results using cursor-based pagination via meta.after_cursor.
func (c *clientImpl) ListGuideMedias() ([]*GuideMedia, error) {
	var all []*GuideMedia
	cursor := ""
	for {
		endpoint := fmt.Sprintf("/api/v2/guide/medias?page[size]=%d", guideMediaListPageSize)
		if cursor != "" {
			endpoint += "&page[after]=" + url.QueryEscape(cursor)
		}
		resp, err := c.doRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		var page guideMediaListResp
		if err := json.Unmarshal([]byte(resp), &page); err != nil {
			return nil, err
		}
		for i := range page.Records {
			all = append(all, &page.Records[i])
		}
		if !page.Meta.HasMore || page.Meta.AfterCursor == "" {
			break
		}
		cursor = page.Meta.AfterCursor
	}
	return all, nil
}
