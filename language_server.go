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

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/jacobdufault/lspc/jsonrpc"
	easyjson "github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jwriter"
	shellwords "github.com/mattn/go-shellwords"
)

// When a language server has been closed it is sent to this channel.
// If this ever blocks the daemon may deadlock.
var languageServerClosed = make(chan *languageServer, 1000)

type responseHandler func(json easyjson.RawMessage)

type languageServer struct {
	cmd *exec.Cmd

	// Directory the language server is running in. Used to determine which
	// language server instance to send a message to.
	directory     string
	nextRequestID RequestID
	onResponse    map[RequestID]responseHandler

	err error

	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func startLanguageServer(bin, directory string, initOpts easyjson.RawMessage) (*languageServer, error) {
	exe, e := shellwords.Parse(bin)
	if e != nil {
		return nil, fmt.Errorf("cannot parse <%s>; error=%s", bin, e.Error())
	}

	ls := languageServer{
		directory:  directory,
		onResponse: make(map[RequestID]responseHandler),
	}

	// Start the binary.
	ls.cmd = exec.Command(exe[0], exe[1:]...)
	ls.cmd.Dir = directory
	ls.stdin, e = ls.cmd.StdinPipe()
	if e != nil {
		return nil, e
	}
	ls.stdout, e = ls.cmd.StdoutPipe()
	if e != nil {
		return nil, e
	}
	ls.stderr, e = ls.cmd.StderrPipe()
	if e != nil {
		return nil, e
	}
	e = ls.cmd.Start()
	if e != nil {
		log.Printf("Got error while starting %s", e.Error())
		return nil, e
	}

	// Handle all process input/output on goroutines.
	go ls.stdoutReader()
	go ls.stderrReader()

	ls.writeInitialize(initOpts)

	return &ls, nil
}

// Write a request, which will have an associated response.
func (l *languageServer) writeRequest(method string, params easyjson.RawMessage, onResponse responseHandler) {

	id := l.nextRequestID
	l.nextRequestID++

	// Use a dummy handler if the user does not care about the result. This
	// prevents log spam from unexpected responses.
	if onResponse == nil {
		onResponse = func(_ easyjson.RawMessage) {}
	}

	l.onResponse[id] = onResponse
	l.rawWriteMsg(method, params, id)
}

func (l *languageServer) writeNotification(method string, params easyjson.RawMessage) {
	l.rawWriteMsg(method, params, -1)
}

// id will only be written to json if it is >= 0
func (l *languageServer) rawWriteMsg(method string, params easyjson.RawMessage, id RequestID) {
	if l.err != nil {
		log.Printf("Attempt to write message while language server has error %s", l.err.Error())
		return
	}

	// content.ID is not written if it is less than 0
	content := JSONRPCHeader{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if _, e := marshalToWriter(content, l.stdin); e != nil {
		l.err = e
		languageServerClosed <- l
	}

	// Uncomment to write the written request to stderr.
	// marshalToWriter(content, os.Stderr)
}

func marshalToWriter(v easyjson.Marshaler, w io.Writer) (written int, err error) {
	jw := jwriter.Writer{}
	jw.Flags = jwriter.NilMapAsEmpty | jwriter.NilSliceAsEmpty
	v.MarshalEasyJSON(&jw)

	// Write header, and then the content
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", jw.Size())
	return jw.DumpTo(w)
}

func toJSON(m easyjson.Marshaler) easyjson.RawMessage {
	jw := jwriter.Writer{}
	jw.Flags = jwriter.NilMapAsEmpty | jwriter.NilSliceAsEmpty
	m.MarshalEasyJSON(&jw)

	r, e := jw.BuildBytes()
	panicIfError(e)
	return r
}

func (l *languageServer) writeInitialize(initOpts easyjson.RawMessage) {
	// Send input.
	l.writeRequest("initialize", toJSON(LsInitializeParams{
		RootURI:               pathToURI(l.directory),
		InitializationOptions: initOpts,
	}), func(json easyjson.RawMessage) {
		log.Print("Got initialize response")
	})
}

func (l *languageServer) stdoutReader() {
	// Build scanner which will process LSP messages.
	scanner := bufio.NewScanner(l.stdout)
	scanner.Split(jsonrpc.SplitFunc)
	// Increase maximum token length; the default is 64 * 1024, which is probably
	// too low.
	const maxScanTokenSize = 1024 * 1024
	scanner.Buffer(make([]byte, 0), maxScanTokenSize)

	for scanner.Scan() {
		header := JSONRPCHeader{}
		header.ID = -1
		header.UnmarshalJSON(scanner.Bytes())
		if header.ID >= 0 {
			if response, has := l.onResponse[header.ID]; has {
				response(header.Params)
			} else {
				log.Printf("No handler for response id %d", header.ID)
			}
		}
	}

	if scanner.Err() != nil {
		l.err = scanner.Err()
	}
	languageServerClosed <- l
}

func (l *languageServer) stderrReader() {
	var buffer [256]byte
	for {
		n, e := l.stderr.Read(buffer[:])
		if e != nil {
			l.err = e
			languageServerClosed <- l
			break
		}
		// For the time being just echo output to our stdout.
		fmt.Printf("stderr: %s", buffer[:n])
	}

	languageServerClosed <- l
}
