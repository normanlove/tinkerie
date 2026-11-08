// Flashie — small byte-stream helpers shared by the AVR and STM32 flashers.
// Made by nrmluvdev.

package flashie

import (
	"fmt"
	"time"
)

type link struct {
	p   Port
	buf []byte
}

func newLink(p Port) *link { return &link{p: p} }

func (k *link) write(b ...byte) error {
	for len(b) > 0 {
		n, err := k.p.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

// readN reads exactly n bytes or fails with errTimeout.
func (k *link) readN(n int, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	tmp := make([]byte, 4096)
	for len(k.buf) < n {
		rem := time.Until(deadline)
		if rem <= 0 {
			return nil, errTimeout
		}
		if rem > 100*time.Millisecond {
			rem = 100 * time.Millisecond
		}
		c, err := k.p.ReadTimeout(tmp, rem)
		if err != nil {
			return nil, err
		}
		k.buf = append(k.buf, tmp[:c]...)
	}
	out := append([]byte{}, k.buf[:n]...)
	k.buf = k.buf[n:]
	return out, nil
}

func (k *link) readByte(timeout time.Duration) (byte, error) {
	b, err := k.readN(1, timeout)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// drain throws away anything arriving within d.
func (k *link) drain(d time.Duration) {
	k.buf = nil
	tmp := make([]byte, 4096)
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		n, err := k.p.ReadTimeout(tmp, 20*time.Millisecond)
		if err != nil || n == 0 {
			if time.Now().After(deadline) {
				break
			}
		}
	}
	k.p.Purge()
}

func expect(got, want byte, what string) error {
	if got != want {
		return fmt.Errorf("%s: unexpected reply 0x%02X (wanted 0x%02X)", what, got, want)
	}
	return nil
}
