package logger

import "launch/internal/pkg/errors"

func LogError(msg string, err errors.Error) {
	if err.IsW() {
		Log.Warn(msg, F("err", err.Error()))
	}
	if err.IsE() {
		Log.Error(msg, F("err", err.Error()))
	}
	if err.IsP() {
		Log.Panic(msg, F("err", err.Error()))
	}
}
