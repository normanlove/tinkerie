//go:build !windows

package scopie

import "errors"

func visaFindUSB() []string { return nil }

type visaLink struct{ rsrc string }

func (v *visaLink) Name() string                   { return v.rsrc }
func (v *visaLink) Open() error                    { return errors.New("VISA is only used on Windows") }
func (v *visaLink) Close()                         {}
func (v *visaLink) Write(b []byte) error           { return errors.New("not connected") }
func (v *visaLink) Read(int) ([]byte, bool, error) { return nil, false, errors.New("not connected") }
