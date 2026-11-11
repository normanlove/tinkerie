//go:build linux

package droid

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"syscall"
	"testing"
)

// A stand-in for the Android app's side of the bridge.
func fakeBridge(t *testing.T, handle func(req map[string]any) (string, *os.File)) {
	name := "tinkerie-test-" + string(rune('a'+os.Getpid()%26))
	l, err := net.Listen("unix", "@"+name)
	if err != nil {
		t.Skip("no abstract sockets here:", err)
	}
	t.Cleanup(func() { l.Close() })
	t.Setenv("TINKERIE_BRIDGE", name)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				line, _ := bufio.NewReader(c).ReadBytes('\n')
				var req map[string]any
				json.Unmarshal(line, &req)
				reply, f := handle(req)
				var oob []byte
				if f != nil {
					oob = syscall.UnixRights(int(f.Fd()))
				}
				c.(*net.UnixConn).WriteMsgUnix([]byte(reply+"\n"), oob, nil)
			}(c)
		}
	}()
}

func TestBridge(t *testing.T) {
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	fakeBridge(t, func(req map[string]any) (string, *os.File) {
		switch req["op"] {
		case "pick":
			return `{"path":"/tmp/x.uf2"}`, nil
		case "usbopen":
			return `{"ok":true}`, w
		case "window":
			if req["url"] != "http://127.0.0.1:1/" || req["app"] != "launcher" {
				return `{"error":"bad request"}`, nil
			}
			return `{"ok":true}`, nil
		}
		return `{"error":"no such thing"}`, nil
	})
	if !Active() {
		t.Fatal("not active")
	}
	if p, err := PickFile("t", []string{".uf2"}); err != nil || p != "/tmp/x.uf2" {
		t.Fatal(p, err)
	}
	if err := OpenWindow("http://127.0.0.1:1/", "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := Call("nope", nil, nil, 0); err == nil || err.Error() != "no such thing" {
		t.Fatal("want error, got", err)
	}
	fd, err := Call("usbopen", nil, nil, 0)
	if err != nil || fd < 0 {
		t.Fatal(fd, err)
	}
	// the handed-over descriptor is the pipe's write end
	syscall.Write(fd, []byte("hi"))
	syscall.Close(fd)
	b := make([]byte, 2)
	r.Read(b)
	if string(b) != "hi" {
		t.Fatal("fd didn't come through:", b)
	}
}
