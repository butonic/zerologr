// Copyright 2019 Jorn Friedrich Dreyer
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// package zerologr defines an implementation of the github.com/go-logr/logr
// interfaces built on top of zerolog (github.com/rs/zerolog).

package zerologr

import (
	"errors"
	"os"

	"github.com/go-logr/logr"
	"github.com/rs/zerolog"
)

// New returns a logr.Logger which is implemented by zerolog.
func New() logr.Logger {
	return NewWithOptions(Options{})
}

// NewWithOptions returns a logr.Logger which is implemented by zerolog.
func NewWithOptions(opts Options) logr.Logger {
	if opts.Logger == nil {
		l := zerolog.New(os.Stderr).With().Timestamp().Logger()
		opts.Logger = &l
	}
	return logger{
		l:      opts.Logger,
		level:  0,
		prefix: opts.Name,
		values: nil,
	}
}

// Options that can be passed to NewWithOptions
type Options struct {
	// Name is an optional name of the logger
	Name string
	// Logger is an instance of zerolog, if nil a default logger is used
	Logger *zerolog.Logger
}

// logger is a logr.Logger that uses zerolog to log.
type logger struct {
	l      *zerolog.Logger
	level  int
	prefix string
	values []interface{}
}

func (l logger) clone() logger {
	out := l
	out.values = copySlice(l.values)
	return out
}

func copySlice(in []interface{}) []interface{} {
	out := make([]interface{}, len(in))
	copy(out, in)
	return out
}

// handleEvent converts a bunch of arbitrary key-value pairs into Zap fields.  It takes
// additional pre-converted Zap fields, for use with automatically attached fields, like
// `error`.
func add(e *zerolog.Event, keysAndVals []interface{}) {
	// a slightly modified version of zap.SugaredLogger.sweetenFields

	// make sure this isn't a mismatched key
	if len(keysAndVals)%2 != 0 {
		e.Interface("args", keysAndVals).
			AnErr("zerologr-err", errors.New("odd number of arguments passed as key-value pairs for logging")).
			Stack()
		return
	}

	for i := 0; i < len(keysAndVals); {
		// process a key-value pair,
		// ensuring that the key is a string
		key, val := keysAndVals[i], keysAndVals[i+1]
		keyStr, isString := key.(string)
		if !isString {
			// if the key isn't a string, log additional error
			e.Interface("invalid key", key).
				AnErr("zerologr-err", errors.New("non-string key argument passed to logging, ignoring all later arguments")).
				Stack()
			return
		}
		e.Interface(keyStr, val)

		i += 2
	}
}

func (l logger) Info(msg string, keysAndVals ...interface{}) {
	if l.Enabled() {
		e := l.l.Info().Int("level", l.level)
		if l.prefix != "" {
			e.Str("name", l.prefix)
		}
		add(e, l.values)
		add(e, keysAndVals)
		e.Msg(msg)
	}
}

func (l logger) Enabled() bool {
	var lvl zerolog.Level
	if l.level < 2 {
		lvl = zerolog.InfoLevel
	} else if l.level < 7 {
		lvl = zerolog.DebugLevel
	} else {
		lvl = zerolog.TraceLevel
	}
	if lvl < zerolog.GlobalLevel() {
		return false
	}
	return true
}

func (l logger) Error(err error, msg string, keysAndVals ...interface{}) {
	e := l.l.Error().Err(err)
	if l.prefix != "" {
		e.Str("name", l.prefix)
	}
	add(e, l.values)
	add(e, keysAndVals)
	e.Msg(msg)
}

func (l logger) V(level int) logr.InfoLogger {
	new := l.clone()
	new.level = level
	return new
}

// WithName returns a new logr.Logger with the specified name appended. zerologr
// uses '/' characters to separate name elements.  Callers should not pass '/'
// in the provided name string, but this library does not actually enforce that.
func (l logger) WithName(name string) logr.Logger {
	new := l.clone()
	if len(l.prefix) > 0 {
		new.prefix = l.prefix + "/"
	}
	new.prefix += name
	return new
}
func (l logger) WithValues(kvList ...interface{}) logr.Logger {
	new := l.clone()
	new.values = append(new.values, kvList...)
	return new
}

var _ logr.Logger = logger{}
var _ logr.InfoLogger = logger{}
