# simple esbuild server

A simple build server using esbuild and fsnotify, serving pages locally over
https. You can do a lot of this with just the esbuild CLI and a tiny amount of 
JS, but I wanted to check out how the Go API works.

Certs and keys are not included, of course, and require some work to generate
and use. If using https locally isn't imprtant to you, consider manually
updating `serve.go::main` to serve over http instead.


## Running

You just have to build and run `serve.go`.

That is, run either `go run serve.go` or something like `go build && ./ses`.


When you build and run the tool, it should build things in `web` once, start an
https server, and then watch files in the `web` directory for changes,
rebuilding if necessary. No HMR or anything is included, so you'll have to
refresh the page yourself.


## Notes

This is just a starter project meant to illustrate how to put together a
small-ish builder and server for the most basic of web projects. The included
web project is a super bare-bones hello world app with some manual styles.

The Go used in this project isn't particularly sophisticated or necessarily
adheres to best practices. `serve.go` is pretty much the whole project at
~330LoC.
