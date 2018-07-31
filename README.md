As a heads up, I'm not likely to work on this project anymore.

# lspc

lspc is a command-line client for the language server protocol ecosystem.
lspc starts and runs language servers and lets you interact with them as if
they were command line tools.

## Installation

```sh
$ go get -u github.com/jacobdufault/lspc
$ lspc # prints out help/usage
# Use lspc help <command-name> for more information, ie, lspc help start
```

## Development status

Still under active work. Managing language servers mostly works, but they
cannot yet be easily interacted with.
