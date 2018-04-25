// Copyright 2018 Jacob Dufault
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonrpc

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"
)

// SplitFunc is a bufio.SplitFunc implementation that splits JsonRPC messages.
func SplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	i := 0

	maybeAddEOFError := func() {
		if atEOF {
			err = errors.New("Expected more content")
		}
	}

	readString := func(content string) {
		for _, c := range content {
			// Not enough input yet; try again later.
			if i >= len(data) {
				maybeAddEOFError()
				return
			}

			if data[i] != byte(c) {
				err = fmt.Errorf("Unexpected token '%c'", data[i])
				return
			}
			i++
		}
	}

	// Read Content-Length:
	if readString("Content-Length: "); err != nil {
		return
	}

	// Read the number.
	digitStart := i
	for {
		// Not enough input yet; try again later.
		if i >= len(data) {
			maybeAddEOFError()
			return
		}

		// Read until we have a number
		if !unicode.IsDigit(rune(data[i])) {
			break
		}

		i++
	}
	contentLength, err := strconv.Atoi(string(data[digitStart:i]))
	if err != nil {
		return
	}

	// Read \r\n\r\n
	if readString("\r\n\r\n"); err != nil {
		return
	}

	// Not enough input yet; try again later.
	if i+contentLength > len(data) {
		maybeAddEOFError()
		return
	}

	// Return the token
	advance = i + contentLength
	token = data[i : i+contentLength]
	return
}
