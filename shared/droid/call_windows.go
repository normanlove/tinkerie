package droid

import (
	"errors"
	"time"
)

// Call is only used on Android.
func Call(op string, args map[string]any, out any, timeout time.Duration) (int, error) {
	return -1, errors.New("not running in the Tinkerie Android app")
}
