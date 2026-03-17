package coolercontrol

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestGetStatusUsesFixedSessionUsernameAndSessionCookie(t *testing.T) {
	t.Helper()

	loginCount := 0
	statusCount := 0
	loginAuthUser := ""
	loginAuthPass := ""
	loginContentType := ""
	loginBody := ""
	var handlerErr error

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/login":
			loginCount++
			loginAuthUser, loginAuthPass, _ = r.BasicAuth()
			loginContentType = r.Header.Get("Content-Type")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				handlerErr = err
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("read login body failed")),
					Request:    r,
				}, nil
			}
			loginBody = string(body)
			if loginAuthUser != coolerControlSessionUsername || loginAuthPass != "secret" {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("invalid credentials")),
					Request:    r,
				}, nil
			}
			header := make(http.Header)
			header.Add("Set-Cookie", "cc-session=ok; Path=/")
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Header:     header,
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    r,
			}, nil
		case "/status":
			statusCount++
			if !strings.Contains(r.Header.Get("Cookie"), "cc-session=ok") {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("unauthorized")),
					Request:    r,
				}, nil
			}
			var body bytes.Buffer
			if err := json.NewEncoder(&body).Encode(coolerControlStatusResponse{Devices: []coolerControlDeviceStatus{}}); err != nil {
				handlerErr = err
				return nil, err
			}
			header := make(http.Header)
			header.Set("Content-Type", "application/json")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     header,
				Body:       io.NopCloser(bytes.NewReader(body.Bytes())),
				Request:    r,
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("not found")),
				Request:    r,
			}, nil
		}
	})
	client := &CoolerControlClient{
		baseURL:  "http://coolercontrol.test",
		password: "secret",
		apiClient: &http.Client{
			Jar:       jar,
			Transport: transport,
		},
		streamClient: &http.Client{
			Jar:       jar,
			Transport: transport,
		},
		readyCh:    make(chan struct{}),
		deviceMeta: make(map[string]coolerControlDeviceNameMap),
	}

	status, err := client.getStatus()
	if err != nil {
		t.Fatalf("getStatus returned error: %v", err)
	}
	if handlerErr != nil {
		t.Fatalf("handler returned error: %v", handlerErr)
	}
	if status == nil {
		t.Fatal("getStatus returned nil status")
	}
	if loginCount != 1 {
		t.Fatalf("expected 1 login request, got %d", loginCount)
	}
	if statusCount != 2 {
		t.Fatalf("expected 2 status requests, got %d", statusCount)
	}
	if loginAuthUser != coolerControlSessionUsername {
		t.Fatalf("expected login user %q, got %q", coolerControlSessionUsername, loginAuthUser)
	}
	if loginAuthPass != "secret" {
		t.Fatalf("expected login password %q, got %q", "secret", loginAuthPass)
	}
	if !strings.HasPrefix(strings.ToLower(loginContentType), "application/json") {
		t.Fatalf("expected JSON login content type, got %q", loginContentType)
	}
	if loginBody != "{}" {
		t.Fatalf("expected login body {}, got %q", loginBody)
	}
}
