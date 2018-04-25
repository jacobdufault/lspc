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
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mailru/easyjson"

	"github.com/urfave/cli"
)

func panicIfError(e error) {
	if e != nil {
		panic(e.Error())
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getSocketFilename() string {
	user := os.Getenv("USER")
	if user == "" {
		user = "all"
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("lspc.%s", user))
}

// Server contains methods which the client can call over rpc.
type Server struct {
	servers []*languageServer
}

func (s *Server) clean() {
	// This is likely not needed now that we properly shutdown language servers with the languageServerClosed channel.
	/*
		i := 0
		for i < len(s.servers) {
			if s.servers[i].err != nil {
				log.Printf("Removing language server %+v in %s", s.servers[i].cmd.Args, s.servers[i].directory)
				s.servers = append(s.servers[:i], s.servers[i+1:]...)
			} else {
				i++
			}
		}
	*/
}

// KeepAlive ensures the server does not shutdown due to inactivity.
func (s *Server) KeepAlive(_ bool, _ *bool) error {
	log.Print("CMD keep-alive")
	timeout := time.Duration(gTimeout) * time.Second
	countdown.Reset(timeout)
	return nil
}

var gShutdown bool

// Kill shuts the server down after a short delay.
// TODO: make kill configurable; kill a specific PID; ls should list the PID to kill (or maybe we want to do `lspc kill 0, lspc kill 1`, etc)
func (s *Server) Kill(_ bool, _ *bool) error {
	log.Print("CMD kill")
	gShutdown = true
	return nil
}

// ServerPid returns the pid of the server.
func (s *Server) ServerPid(_ bool, pid *int) error {
	log.Print("CMD server-pid")
	*pid = os.Getpid()
	return nil
}

// Ls lists running servers.
func (s *Server) Ls(_ bool, servers *[]string) error {
	log.Print("CMD ls")
	s.clean()
	for _, server := range s.servers {
		*servers = append(*servers, fmt.Sprintf("%+v in %s", server.cmd.Args, server.directory))
	}
	return nil
}

// StartArgs holds arguments for Start.
type StartArgs struct {
	Bin       string
	Directory string
	InitOpts  easyjson.RawMessage
}

// Start runs a new language server.
func (s *Server) Start(args StartArgs, _ *bool) error {
	log.Printf("CMD start %s in %s", args.Bin, args.Directory)

	ls, err := startLanguageServer(args.Bin, args.Directory, args.InitOpts)
	if err != nil {
		return err
	}

	s.servers = append(s.servers, ls)
	return nil
}

var countdown *time.Timer

func daemonMainLoop() {
	if gSocket == "" {
		gSocket = getSocketFilename()
	}

	// Register RPC
	server := new(Server)
	rpc.Register(server)

	// Open the socket.
	if !gDisableRemoveSocket {
		if err := os.Remove(gSocket); err == nil {
			log.Printf("Removed existing socket")
		}
	}
	log.Printf("Opening socket at %s", gSocket)
	listener, err := net.Listen("unix", gSocket)
	panicIfError(err)
	defer func() {
		panicIfError(listener.Close())
	}()

	// goroutine that listens for new connections.
	conn := make(chan net.Conn)
	go func() {
		for {
			c, e := listener.Accept()
			if e != nil {
				if !gShutdown {
					log.Printf("%s", e.Error())
				}
				gShutdown = true
				return
			}
			conn <- c
		}
	}()

	// Main loop. Handles incoming requests.
	timeout := time.Duration(gTimeout) * time.Second
	countdown = time.NewTimer(timeout)
loop:
	for {
		select {
		case c := <-conn:
			rpc.ServeConn(c)

			if gShutdown {
				break loop
			}

			countdown.Reset(timeout)
			runtime.GC()

		case closed := <-languageServerClosed:
			i := 0
			for i < len(server.servers) {
				if server.servers[i] == closed {
					if closed.err == nil {
						log.Printf("Language server %+v in %s has closed", closed.cmd.Args, closed.directory)
					} else {
						log.Printf("Language server %+v in %s has closed (err=%s)", closed.cmd.Args, closed.directory, closed.err.Error())
					}
					server.servers = append(server.servers[:i], server.servers[i+1:]...)
				} else {
					i++
				}
			}

		case <-countdown.C:
			break loop
		}
	}
}

func ensureDaemon() {
	// FIXME: add logic for this

	// Early-exit if the socket exists already.
	if fileExists(gSocket) {
		return
	}

	log.Printf("Starting daemon")

	path, err := os.Executable()
	panicIfError(err)

	p := exec.Command(path, "-socket", gSocket, "daemon")
	err = p.Start()
	panicIfError(err)
}

func doRPC(serviceMethod string, args interface{}, reply interface{}) {
	// Fetch the socket filename if not specified
	if len(gSocket) == 0 {
		gSocket = getSocketFilename()
	}

	// Try to connect. If it fails, start a server.
	conn, e := rpc.Dial("unix", gSocket)
	if e != nil {
		ensureDaemon()

		// Try to connect again if we after waiting a bit for the server to start.
		// FIXME: is there a more robust approach here?
		time.Sleep(time.Millisecond * 250)

		conn, e = rpc.Dial("unix", gSocket)
		if e != nil {
			fmt.Printf("Unable to connect to socket: %s\n", e.Error())
			os.Exit(2)
		}
	}

	e = conn.Call(serviceMethod, args, reply)
	conn.Close()

	if e != nil {
		fmt.Printf("error during rpc: %s", e.Error())
		os.Exit(1)
	}
}

var gSocket string
var gDisableRemoveSocket bool
var gTimeout int

func main() {
	app := cli.NewApp()
	app.Name = "lspc"
	app.Usage = "language server protocol client"
	app.Description =
		`lspc manages language servers and provides convenient APIs to interact with
   those language servers`

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "socket",
			Usage:       "Path to a socket that lspc will use to communicate with the daemon.",
			EnvVar:      "LSPC_SOCKET",
			Destination: &gSocket,
		},
		cli.BoolFlag{
			Name:        "disable-remove-socket",
			Usage:       "Do not try to remove the socket if it already exists.",
			EnvVar:      "LSPC_REMOVE_SOCKET",
			Destination: &gDisableRemoveSocket,
		},
		cli.IntFlag{
			Name:        "timeout",
			Usage:       "Seconds until the server shuts down. Reset whenever the server receives a request",
			EnvVar:      "LSPC_TIMEOUT",
			Value:       60 * 30,
			Destination: &gTimeout,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:        "keep-alive",
			Description: "Send a ping to the server to make sure it does not shut down",
			Action: func(c *cli.Context) error {
				doRPC("Server.KeepAlive", false, nil)
				return nil
			},
		},
		{
			Name:        "kill",
			Description: "Shut the server down",
			Action: func(c *cli.Context) error {
				doRPC("Server.Kill", false, nil)
				return nil
			},
		},
		{
			Name:        "server-pid",
			Description: "Shut the server down",
			Action: func(c *cli.Context) error {
				var pid int
				doRPC("Server.ServerPid", false, &pid)
				println(pid)
				return nil
			},
		},
		{
			Name:        "ls",
			Description: "List all running language servers",
			Action: func(c *cli.Context) error {
				var servers []string
				doRPC("Server.Ls", false, &servers)
				for _, server := range servers {
					println(server)
				}
				return nil
			},
		},
		{
			Name:      "start",
			Usage:     "start a new language server",
			UsageText: "lspc start <bin> <project-dir> [<init>]",
			Description: `<bin> can be a quoted string which will be parsed as shell words, ie,
   "cquery --log-file log.txt" will run cquery with the arguments [--log-file, log.txt]

   <project-dir> is the directory containing the content that the language server will
   analyze

   <init> can be a raw json literal passed to the language server in the initialization
	 message, ex, '{"cacheDirectory": "/ssd/cquery_cache/"}'. Defaults to {}

   Example:
    $ lspc start "cquery --log-all-to-stderr" /work/chrome '{"cacheDirectory": "/ssd/cquery_cache"}'`,
			Action: func(c *cli.Context) error {
				if c.NArg() != 2 && c.NArg() != 3 {
					return cli.ShowCommandHelp(c, "start")
				}

				init := c.Args().Get(2)
				if init == "" {
					init = "{}"
				}
				args := StartArgs{
					Bin:       c.Args().Get(0),
					Directory: c.Args().Get(1),
					InitOpts:  easyjson.RawMessage(init),
				}

				doRPC("Server.Start", args, nil)
				return nil
			},
		},
		{
			Name:        "daemon",
			Usage:       "run the lspc daemon",
			UsageText:   "lspc daemon",
			Description: "Run the lspc daemon. In typical operation lspc will start the daemon for you.",
			Action: func(c *cli.Context) error {
				daemonMainLoop()
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
