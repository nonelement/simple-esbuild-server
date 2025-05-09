# simple esbuild server

A super simple build server using esbuild, serving pages locally over
https. This template uses Preact, but you can switch it to React pretty easily.

Certs and keys are not included, of course, and require some work to generate
and use. If using https locally isn't imprtant to you, consider manually
updating `serve.go::main` to serve over http instead.


## Running

You just have to build and run `serve.go`.

That is, run either `go run serve.go` or something like `go build && ./ses`.

When you start `ses`, it should build dist, serve up the results, and then
rebuild whenever there are chnages to files in `web/`. No HMR or anything is
included, so you'll have to refresh the page yourself.


## Notes

This is _very_ similar to the configuration you'd use if you were using the
Node.js interface, so the use of this depends mostly on whether you want to
add, say, an API in Go versus JS.
