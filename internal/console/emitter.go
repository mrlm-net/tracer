package console

import (
	"os"

	eventpkg "github.com/mrlm-net/tracer/pkg/event"
)

// makeEmitter returns an event.Emitter and optionally a BufferingEmitter
// when outputChoice == "html".
func makeEmitter(outputChoice string, stdout *os.File) (eventpkg.Emitter, *eventpkg.BufferingEmitter) {
	if outputChoice == "html" {
		be := eventpkg.NewBufferingEmitter()
		return be, be
	}
	return eventpkg.NewStdoutEmitter(stdout, true, true), nil
}
