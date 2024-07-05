package httplog

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func init() {
	http.DefaultTransport = DefaultTransport
}

type Transport struct {
	Transport   http.RoundTripper
	LogRequest  func(req *http.Request)
	LogResponse func(resp *http.Response)
}

type contextRequestKey struct {
	name string
}

var ContextRequestKey = contextRequestKey{"httplog.zendesk.api"}

var DefaultTransport = &Transport{
	Transport: http.DefaultTransport,
}

func DefaultLogRequest(req *http.Request) {
	log.Printf("----> [%s] %s", req.Method, req.URL)
}

func DefaultLogResponse(res *http.Response) {
	ctx := res.Request.Context()
	if requestedAt, ok := ctx.Value(ContextRequestKey).(time.Time); ok {
		d := fmt.Sprintf("%dms", time.Duration(time.Since(requestedAt)).Milliseconds())
		pd, _ := time.ParseDuration(d)
		log.Printf("<---- [%s] %s (%s)", res.Status, res.Request.URL, pd)
	} else {
		log.Printf("<---- [%s] %s", res.Status, res.Request.URL)
	}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), ContextRequestKey, time.Now())
	req = req.WithContext(ctx)

	t.logRequest(req)
	res, err := t.transport().RoundTrip(req)
	if err != nil {
		return res, err
	}
	t.logResponse(res)

	return res, err
}

func (t *Transport) logRequest(req *http.Request) {
	if t.LogRequest != nil {
		t.LogRequest(req)
	} else {
		DefaultLogRequest(req)
	}
}

func (t *Transport) logResponse(res *http.Response) {
	if t.LogResponse != nil {
		t.LogResponse(res)
	} else {
		DefaultLogResponse(res)
	}
}

func (t *Transport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}
