package fahapi

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testSysApID = "00000000-0000-0000-0000-000000000000"

// serveApi points the library at a test server and returns the requests it saw.
func serveApi(t *testing.T, handler http.HandlerFunc) *[]*http.Request {
	t.Helper()
	quietApi(t)

	var seen []*http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	ConfigureApi(strings.TrimPrefix(srv.URL, "http://"), "user", "secret", nil, nil, logger, 0)
	return &seen
}

// RFC 7617 has no colon after the scheme. Sending one made current SysAP
// firmware answer 401 to every request.
func TestAuthorizationHeaderFormat(t *testing.T) {
	seen := serveApi(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"`+testSysApID+`":{"sysapName":"test","devices":{}}}`)
	})

	if _, err := GetConfiguration(); err != nil {
		t.Fatalf("GetConfiguration: %v", err)
	}

	got := (*seen)[0].Header.Get("Authorization")
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:secret"))
	if got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
	if strings.HasPrefix(got, "Basic:") {
		t.Error("the scheme still carries a colon")
	}
}

func TestGetConfiguration(t *testing.T) {
	serveApi(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"`+testSysApID+`":{"sysapName":"my sysap","devices":{"DEV1":{"displayName":"one"}}}}`)
	})

	sysap, err := GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration: %v", err)
	}
	if Str(sysap.SysapName) != "my sysap" {
		t.Errorf("SysapName = %q", Str(sysap.SysapName))
	}
	if len(sysap.Devices) != 1 {
		t.Errorf("got %d devices, want 1", len(sysap.Devices))
	}
}

func TestGetDatapoint(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		serveApi(t, func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, `{"`+testSysApID+`":{"values":["42"]}}`)
		})
		got, err := GetDatapoint(testSysApID, "DEV1", "ch0000", "odp0000")
		if err != nil {
			t.Fatalf("GetDatapoint: %v", err)
		}
		if got != "42" {
			t.Errorf("value = %q, want 42", got)
		}
	})

	// Indexing Values[0] without a length check used to panic here.
	t.Run("no values", func(t *testing.T) {
		serveApi(t, func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, `{"`+testSysApID+`":{"values":[]}}`)
		})
		if _, err := GetDatapoint(testSysApID, "DEV1", "ch0000", "odp0000"); err == nil {
			t.Error("empty value list accepted")
		}
	})
}

func TestPutDatapoint(t *testing.T) {
	seen := serveApi(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "1" {
			t.Errorf("body = %q, want 1", body)
		}
		io.WriteString(w, `{"`+testSysApID+`":{"result":"OK"}}`)
	})

	ok, err := PutDatapoint(testSysApID, "DEV1", "ch0000", "idp0000", "1")
	if err != nil {
		t.Fatalf("PutDatapoint: %v", err)
	}
	if !ok {
		t.Error("PutDatapoint reported failure for an OK result")
	}

	req := (*seen)[0]
	if req.Method != http.MethodPut {
		t.Errorf("method = %s, want PUT", req.Method)
	}
	if ct := req.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

// The SysAP puts usable diagnostics in the body of a failed request.
func TestErrorIncludesResponseBody(t *testing.T) {
	serveApi(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "device is locked")
	})

	_, err := GetConfiguration()
	if err == nil {
		t.Fatal("expected an error for status 403")
	}
	if !strings.Contains(err.Error(), "device is locked") {
		t.Errorf("error %q does not carry the response body", err)
	}
}

func TestPutVirtualDevice(t *testing.T) {
	serveApi(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"`+testSysApID+`":{"devices":{"6000ABCDEF01":{"serial":"mine"}}}}`)
	})

	serial, err := PutVirtualDevice(testSysApID, "mine", &VirtualDevice{
		Type:       VirtualDeviceType_SwitchingActuator,
		Properties: VirtualDeviceProperties{Displayname: "test", Ttl: "180"},
	})
	if err != nil {
		t.Fatalf("PutVirtualDevice: %v", err)
	}
	if serial != "6000ABCDEF01" {
		t.Errorf("serial = %q, want 6000ABCDEF01", serial)
	}
}
