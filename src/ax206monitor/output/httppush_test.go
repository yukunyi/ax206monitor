package output

import (
	"bytes"
	"encoding/base64"
	"image"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestHTTPPushBuildMultipartPayload(t *testing.T) {
	handler := NewHTTPPushOutputHandler(OutputConfig{
		Type:        TypeHTTPPush,
		URL:         "http://127.0.0.1/upload",
		Method:      "PUT",
		BodyMode:    "multipart",
		Format:      "jpeg_baseline",
		Quality:     77,
		ContentType: "image/jpeg",
		FileField:   "image",
		FileName:    "latest.jpg",
		FormFields: []HTTPKeyValue{
			{Key: "device", Value: "ax206"},
			{Key: "mode", Value: "preview"},
		},
		TimeoutMS: 3000,
	}, TypeHTTPPush)
	defer handler.Close()

	payload, contentType, err := handler.buildRequestPayload([]byte("raw-image"), "image/jpeg")
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}
	if !strings.HasPrefix(contentType, "multipart/form-data;") {
		t.Fatalf("unexpected content type: %q", contentType)
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("parse content type: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("unexpected media type: %q", mediaType)
	}
	reader := multipart.NewReader(bytes.NewReader(payload), params["boundary"])

	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("first part: %v", err)
	}
	if part.FormName() != "device" {
		t.Fatalf("unexpected first field name: %q", part.FormName())
	}

	part, err = reader.NextPart()
	if err != nil {
		t.Fatalf("second part: %v", err)
	}
	if part.FormName() != "mode" {
		t.Fatalf("unexpected second field name: %q", part.FormName())
	}

	part, err = reader.NextPart()
	if err != nil {
		t.Fatalf("file part: %v", err)
	}
	if part.FormName() != "image" {
		t.Fatalf("unexpected file field: %q", part.FormName())
	}
	if part.FileName() != "latest.jpg" {
		t.Fatalf("unexpected file name: %q", part.FileName())
	}
	if part.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("unexpected file content type: %q", part.Header.Get("Content-Type"))
	}
}

func TestHTTPPushDoRequestHonorsMethodAuthHeadersAndSuccessCodes(t *testing.T) {
	var seenMethod string
	var seenContentType string
	var seenAuth string
	var seenCustomHeader string
	var seenBody string

	handler := NewHTTPPushOutputHandler(OutputConfig{
		Type:        TypeHTTPPush,
		URL:         "http://127.0.0.1/test",
		Method:      "PATCH",
		BodyMode:    "binary",
		ContentType: "application/octet-stream",
		Headers: []HTTPKeyValue{
			{Key: "X-Test", Value: "ok"},
		},
		AuthType:     "basic",
		AuthUsername: "demo",
		AuthPassword: "secret",
		SuccessCodes: []int{201},
		TimeoutMS:    3000,
	}, TypeHTTPPush)
	defer handler.Close()
	handler.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		seenMethod = r.Method
		seenContentType = r.Header.Get("Content-Type")
		seenAuth = r.Header.Get("Authorization")
		seenCustomHeader = r.Header.Get("X-Test")
		body, _ := io.ReadAll(r.Body)
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusCreated,
			Status:     "201 Created",
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})

	if err := handler.doRequest([]byte("payload"), "image/jpeg"); err != nil {
		t.Fatalf("do request: %v", err)
	}
	if seenMethod != "PATCH" {
		t.Fatalf("unexpected method: %q", seenMethod)
	}
	if seenContentType != "application/octet-stream" {
		t.Fatalf("unexpected content type: %q", seenContentType)
	}
	if seenCustomHeader != "ok" {
		t.Fatalf("unexpected custom header: %q", seenCustomHeader)
	}
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("demo:secret"))
	if seenAuth != expectedAuth {
		t.Fatalf("unexpected auth header: %q", seenAuth)
	}
	if seenBody != "payload" {
		t.Fatalf("unexpected body: %q", seenBody)
	}
}

func TestHTTPPushEncodeFrameHonorsBaselineFormat(t *testing.T) {
	handler := NewHTTPPushOutputHandler(OutputConfig{
		Type:      TypeHTTPPush,
		Format:    "jpeg_baseline",
		Quality:   72,
		TimeoutMS: 3000,
	}, TypeHTTPPush)
	defer handler.Close()

	frame := NewOutputFrame(image.NewRGBA(image.Rect(0, 0, 4, 4)))
	data, contentType, err := handler.encodeFrame(frame)
	if err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected encoded jpeg bytes")
	}
	if contentType != "image/jpeg" {
		t.Fatalf("unexpected content type: %q", contentType)
	}
}
