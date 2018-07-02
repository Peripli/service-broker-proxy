package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	logSourceField          = "logSource"
	logSourceSetMarkerField = "@logSourceSetBy"
	logrusPackage           = "github.com/sirupsen/logrus"
	arkPackage              = "github.com/heptio/ark/"
	arkPackageLen           = len(arkPackage)
)

// LogLocationHook is a logrus hook that attaches location information
// to log entries, i.e. the file and line number of the logrus log call.
// This hook is designed for use in both the Ark server and Ark plugin
// implementations. When triggered within a plugin, a marker field will
// be set on the log entry indicating that the location came from a plugin.
// The Ark server instance will not overwrite location information if
// it sees this marker.
type LogLocationHook struct {
	loggerName string
}

// WithLoggerName gives the hook a name to use when setting the marker field
// on a log entry indicating the location has been recorded by a plugin. This
// should only be used when setting up a hook for a logging used in a plugin.
func (h *LogLocationHook) WithLoggerName(name string) *LogLocationHook {
	h.loggerName = name
	return h
}

func (h *LogLocationHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *LogLocationHook) Fire(entry *logrus.Entry) error {
	pcs := make([]uintptr, 64)

	// skip 2 frames:
	//   runtime.Callers
	//   github.com/heptio/ark/pkg/util/logging/(*LogLocationHook).Fire
	n := runtime.Callers(2, pcs)

	// re-slice pcs based on the number of entries written
	frames := runtime.CallersFrames(pcs[:n])

	// now traverse up the call stack looking for the first non-logrus
	// func, which will be the logrus invoker
	var (
		frame runtime.Frame
		more  = true
	)

	for more {
		frame, more = frames.Next()

		if strings.Contains(frame.File, logrusPackage) {
			continue
		}

		// set the marker field if we're within a plugin indicating that
		// the location comes from the plugin.
		if h.loggerName != "" {
			entry.Data[logSourceSetMarkerField] = h.loggerName
		}

		// record the log statement location if we're within a plugin OR if
		// we're in Ark server and not logging something that has the marker
		// set (which would indicate the log statement is coming from a plugin).
		if h.loggerName != "" || getLogSourceSetMarker(entry) == "" {
			file := removeArkPackagePrefix(frame.File)

			entry.Data[logSourceField] = fmt.Sprintf("%s:%d", file, frame.Line)
		}

		// if we're in the Ark server, remove the marker field since we don't
		// want to record it in the actual log.
		if h.loggerName == "" {
			delete(entry.Data, logSourceSetMarkerField)
		}

		break
	}

	return nil
}

func getLogSourceSetMarker(entry *logrus.Entry) string {
	nameVal, found := entry.Data[logSourceSetMarkerField]
	if !found {
		return ""
	}

	if name, ok := nameVal.(string); ok {
		return name
	}

	return fmt.Sprintf("%s", nameVal)
}

func removeArkPackagePrefix(file string) string {
	if index := strings.Index(file, arkPackage); index != -1 {
		// strip off .../github.com/heptio/ark/ so we just have pkg/...
		return file[index+arkPackageLen:]
	}

	return file
}
