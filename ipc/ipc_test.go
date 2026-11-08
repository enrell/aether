package ipc

import (
	"strings"
	"testing"
)

type echoDispatcher struct{}

func (echoDispatcher) HandleIPC(req Request) Response {
	if req.Cmd == "fail" {
		return Response{OK: false, Error: "failed"}
	}
	return Response{OK: true, Mode: req.Mode, Palette: req.Palette}
}

func TestServerRoundTrip(t *testing.T) {
	path := testSocketPath(t)
	srv, err := NewServer(path, echoDispatcher{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	resp, err := Send(path, Request{Cmd: "status", Mode: "pastel", Palette: []string{"#000000"}})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK || resp.Mode != "pastel" || len(resp.Palette) != 1 {
		t.Errorf("response = %+v", resp)
	}
	resp, err = Send(path, Request{Cmd: "fail"})
	if err != nil || resp.OK || resp.Error != "failed" {
		t.Errorf("failure response = %+v, %v", resp, err)
	}

	if _, err := NewServer(path, echoDispatcher{}); err == nil {
		t.Error("a second server claimed the same address")
	}
}

func TestSendWhenNotRunning(t *testing.T) {
	path := testSocketPath(t)
	_, err := Send(path, Request{Cmd: "status"})
	if err == nil || !strings.Contains(err.Error(), "aether is not running") {
		t.Errorf("Send error = %v", err)
	}
}

func TestServerCloseReleasesAddress(t *testing.T) {
	path := testSocketPath(t)
	srv, err := NewServer(path, echoDispatcher{})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Close(); err != nil {
		t.Fatal(err)
	}
	srv, err = NewServer(path, echoDispatcher{})
	if err != nil {
		t.Fatalf("address still taken after Close: %v", err)
	}
	srv.Close()
}
