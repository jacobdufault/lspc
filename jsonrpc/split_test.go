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
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadBadHeader(t *testing.T) {
	input := "foobar"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(SplitFunc)
	scanner.Scan()
	assert.Error(t, scanner.Err())
}

func TestReadBadHeaderLength(t *testing.T) {
	input := "Content-Length: aa\r\n\r\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(SplitFunc)
	scanner.Scan()
	assert.Error(t, scanner.Err())
}

func TestReadBadHeaderSeparator(t *testing.T) {
	input := "Content-Length: 5\r\r\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(SplitFunc)
	scanner.Scan()
	assert.Error(t, scanner.Err())
}

func TestReadEmptyMessage(t *testing.T) {
	input := "Content-Length: 0\r\n\r\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(SplitFunc)
	scanner.Scan()
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "", scanner.Text())
}

func TestReadSmallMessage(t *testing.T) {
	input := "Content-Length: 3\r\n\r\nabc"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(SplitFunc)
	scanner.Scan()
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "abc", scanner.Text())
}

func TestReadMultipleMessages(t *testing.T) {
	input := "Content-Length: 3\r\n\r\nabc" +
		"Content-Length: 5\r\n\r\n12345" +
		"Content-Length: 2000\r\n\r\nabc" // Not enough content.

	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(SplitFunc)

	scanner.Scan()
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "abc", scanner.Text())

	scanner.Scan()
	assert.NoError(t, scanner.Err())
	assert.Equal(t, "12345", scanner.Text())

	scanner.Scan()
	assert.Error(t, scanner.Err())
}
