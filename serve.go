package main

import (
  "os"
	"github.com/joho/godotenv"
  esbuild "github.com/evanw/esbuild/pkg/api"
)


// Environment keys
const ENV_CERT = "CERT_FILE"
const ENV_KEY = "KEY_FILE"


// Paths for assets.
const WEBDIR string = "./web"
const DISTDIR string = "./dist"


func buildContext() (esbuild.BuildContext, *esbuild.ContextError) {
	return esbuild.Context(esbuild.BuildOptions{
		EntryPoints: []string{"web/main.tsx", "web/styles.css", "web/index.html"},
		Outdir: "dist",
		Bundle: true,
		Sourcemap: esbuild.SourceMapExternal,
		Loader: map[string]esbuild.Loader{".html": esbuild.LoaderCopy},
		JSX: esbuild.JSXAutomatic,
		// Changes the following if you want to switch to React
		JSXFactory: "h",
		JSXFragment: "Fragment",
		JSXImportSource: "preact",
		Write: true,
		LogLevel: esbuild.LogLevelInfo,
	})
}


func main() {
	godotenv.Load()

	cert := os.Getenv(ENV_CERT)
	key := os.Getenv(ENV_KEY)

	ctx, ctxErr := buildContext()
	defer ctx.Dispose()

	if ctxErr != nil {
		panic(ctxErr)
	}

	watchErr := ctx.Watch(esbuild.WatchOptions{})

	if watchErr != nil {
		panic(watchErr)
	}

	_, serveErr := ctx.Serve(esbuild.ServeOptions{
		Servedir: "dist",
		Port: 443,
		Keyfile: key,
		Certfile: cert,
	});

	if serveErr != nil {
		panic(serveErr)
	}

	<-make(chan struct{})
}
