//go:build !darwin

package transcription

import "errors"

func NewApple() (Engine, error) { return nil, errors.New("Apple speech is only available on macOS") }
