//go:build !windows

package droid

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
	"time"
)

// Call sends one request and decodes the reply into out (may be nil). It
// returns the file descriptor the reply carried, or -1.
func Call(op string, args map[string]any, out any, timeout time.Duration) (int, error) {
	name := os.Getenv("TINKERIE_BRIDGE")
	if name == "" {
		return -1, errors.New("not running in the Tinkerie Android app")
	}
	c, err := net.DialTimeout("unix", "@"+name, 3*time.Second)
	if err != nil {
		return -1, fmt.Errorf("couldn't reach the Tinkerie app: %v", err)
	}
	defer c.Close()
	if timeout > 0 {
		c.SetDeadline(time.Now().Add(timeout))
	}
	req := map[string]any{"op": op, "app": App(), "pid": os.Getpid()}
	for k, v := range args {
		req[k] = v
	}
	b, _ := json.Marshal(req)
	if _, err := c.Write(append(b, '\n')); err != nil {
		return -1, err
	}
	uc := c.(*net.UnixConn)
	var line bytes.Buffer
	fd := -1
	buf, oob := make([]byte, 64<<10), make([]byte, syscall.CmsgSpace(4*4))
	for !bytes.Contains(line.Bytes(), []byte{'\n'}) {
		n, oobn, _, _, err := uc.ReadMsgUnix(buf, oob)
		if oobn > 0 && fd < 0 {
			if msgs, err := syscall.ParseSocketControlMessage(oob[:oobn]); err == nil {
				for _, m := range msgs {
					if fds, err := syscall.ParseUnixRights(&m); err == nil && len(fds) > 0 {
						fd = fds[0]
						for _, extra := range fds[1:] {
							syscall.Close(extra)
						}
					}
				}
			}
		}
		line.Write(buf[:n])
		if err != nil {
			if line.Len() > 0 && errors.Is(err, io.EOF) {
				break
			}
			if fd >= 0 {
				syscall.Close(fd)
			}
			return -1, err
		}
		if n == 0 && oobn == 0 {
			break
		}
	}
	var r struct {
		Error string `json:"error"`
	}
	raw, _ := bufio.NewReader(&line).ReadBytes('\n')
	if err := json.Unmarshal(raw, &r); err != nil {
		if fd >= 0 {
			syscall.Close(fd)
		}
		return -1, fmt.Errorf("odd reply from the Tinkerie app: %v", err)
	}
	if r.Error != "" {
		if fd >= 0 {
			syscall.Close(fd)
		}
		return -1, errors.New(r.Error)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fd, err
		}
	}
	return fd, nil
}
