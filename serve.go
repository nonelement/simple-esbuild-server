package main

import (
	"context"
  "fmt"
  "io"
  "io/fs"
  "log"
  "net/http"
  "os"
  "path"
  "slices"
  "strings"
	"time"
	"github.com/joho/godotenv"
  esbuild "github.com/evanw/esbuild/pkg/api"
	fsnotify "github.com/fsnotify/fsnotify"
)


// Environment keys
const ENV_CERT = "CERT_FILE"
const ENV_KEY = "KEY_FILE"


// Paths for assets.
const WEBDIR string = "./web"
const DISTDIR string = "./dist"


// Helper Struct: Combines a file path and some metadata for handling.
type PathDirPair struct {
  Path string
  Entry fs.DirEntry
}


// Helper Struct: Used primarily to bundle data from a copy operation.
type FSOp struct {
	Path string
	BytesCopied int64
	Err error
}


// util. copy a file at src path, to dest folder.
func copyFile(src, dest string) (int64, error) {
  var sep string = string(os.PathSeparator)
  sourceStat, err := os.Stat(src)
  if err != nil {
    return 0, err
  }

  if !sourceStat.Mode().IsRegular() {
    return 0, fmt.Errorf("% is not a regular file", src)
  }

  s, err := os.Open(src)
  if err != nil {
    return 0, err
  }
  defer s.Close()
  // ok to here
  // Split into path segments based on os separator
  parts := strings.Split(src, sep)
  // Files are assumed to originate from "web/.." dir, so lets strip it
  subdir := path.Dir(strings.Join(parts[1:], sep))
  // Generate dest path from subdir, which is where the copied files will go
  destSubdir := path.Join(dest, subdir)

  err = os.MkdirAll(destSubdir, 0700)
  if err != nil {
    return 0, fmt.Errorf("unable to create interstitial folder: %s, %s", destSubdir, err)
  }
  newFile := path.Join(destSubdir, path.Base(src))
  d, err := os.Create(newFile)
  if err != nil {
    return 0, err
  }
  defer d.Close()
  nBytes, err := io.Copy(d, s)
  return nBytes, err
}

// Wrapper that runs copyFile on all file paths passed in.
func copyFiles(files []string) []FSOp {
	var ops []FSOp
  for i := 0; i < len(files); i++ {
    n, err := copyFile(files[i], DISTDIR)
		ops = append(ops, FSOp{files[i], n, err})
  }
	return ops
}

// util. clears dist folder.
// will remove dist folder and recreate it.
func clearDist() error {
  err := os.RemoveAll(DISTDIR);
  if err != nil {
    return err
  }
  err = os.Mkdir(DISTDIR, 0770);
  if err != nil {
    return err
  }
  return nil
}

// util. get a file's extension, e.g. "css" for "syles.css"
func getExt(f string) string {
  parts := strings.Split(f, ".")
  return parts[len(parts)-1:][0]
}


// Get files in the web subdirectory by walking, while ignoring some files.
func getWebFiles() []PathDirPair {
	var ignored = []string{"node_modules", "package.json", "pnpm-lock.yaml"}
  var results []PathDirPair
  webDir := os.DirFS(WEBDIR)
  fs.WalkDir(webDir, ".", func(p string, d fs.DirEntry, err error) error {
		// Skip entries if they have a path segment we want to ignore.
		for i := 0; i < len(ignored); i++ {
			if strings.Contains(p, ignored[i]) {
				return nil
			}
		}
    if err != nil {
      log.Fatal(err)
    }
    results = append(results, PathDirPair{path.Join(WEBDIR, p), d})
    return nil
  })
  return results
}


// Partition directory contents into copyable and skippable.
func filterCopyableFiles(files []PathDirPair) ([]string, []string) {
	// Dont copy, since these are built.
  buildFiles := []string{"ts", "tsx", "js"}
  var toCopy []string
  var skipping []string
  for i := 0; i < len(files); i++ {
    var f = files[i]
    if !f.Entry.IsDir() {
      ext := getExt(f.Path)
      if !slices.Contains(buildFiles, ext) {
        toCopy = append(toCopy, f.Path)
      } else {
        skipping = append(skipping, f.Path)
      }
    }
  }
  return toCopy, skipping
}


// Top level asset copying fn.
func copyAssets() []FSOp {
  pdp := getWebFiles()
  toCopy, _ := filterCopyableFiles(pdp)
	return copyFiles(toCopy)
}


// Low level JS building fn.
func buildJS() esbuild.BuildResult {
  result := esbuild.Build(esbuild.BuildOptions{
    EntryPoints: []string{"web/main.tsx"},
		JSX: esbuild.JSXAutomatic,
		JSXFactory: "h",
		JSXFragment: "Fragment",
		JSXImportSource: "preact",
    Outfile: "dist/main.js",
		Sourcemap: esbuild.SourceMapExternal,
    Bundle: true,
    Write: true,
    //LogLevel: esbuild.LogLevelInfo,
  })

  return result
}


// Top level JS building fn.
func makeWeb() ([]FSOp, esbuild.BuildResult) {
  clearDist()
	return copyAssets(), buildJS()
}


// Utility for displaying build results from copy and esbuild.
func displayResults(ops []FSOp, result esbuild.BuildResult) () {
	for i := 0; i < len(ops); i++ {
		if ops[i].Err != nil {
      fmt.Printf("x failed %s: %s\n", ops[i].Path, ops[i].Err)
    } else {
      fmt.Printf("o copied %s, %d bytes\n", ops[i].Path, ops[i].BytesCopied)
    }
	}

	if len(result.Errors) > 0 {
  	fmt.Println(result.Errors)
	} else {
		fmt.Println("build ok.")
	}
}


// Watches the web directory for changes and then rebuilds assets on fs ops.
func watchWeb() {
	// Set up fs watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Ops we build on.
	buildOps := []fsnotify.Op{
		fsnotify.Create,
		fsnotify.Write,
		fsnotify.Rename,
		fsnotify.Remove,
	}

	// Build callback that gets called on fs events
	build := func(ctx context.Context) {
		delay := time.NewTimer(time.Second)
		select {
		case <-delay.C:
			fmt.Printf("// building...");
			ops, result := makeWeb()
			for i, op := range ops {
				if op.Err != nil {
					fmt.Printf("x failed %s: %s\n", ops[i].Path, ops[i].Err)
				}
			}
			if len(result.Errors) > 0 {
				fmt.Printf(" failed\n")
				for _, err := range result.Errors {
					fmt.Println(err)
				}
			} else {
				fmt.Printf(" ok\n")
			}
		case <-ctx.Done():
			return
		}
	}

	// Event processing goroutine
	go func() {
		// Declare context, cancel fn here to make them available across loops.
		var (
			ctx context.Context
			cancel context.CancelFunc
		)
		for {
			select {
				case event, ok := <- watcher.Events:
					if !ok {
						return
					}
					// log.Println("event: " , event)

					// Only build on certain events
					if slices.Contains(buildOps, event.Op) {
						if cancel != nil {
							cancel()
						}
						ctx, cancel = context.WithCancel(context.Background())
						go build(ctx)
					}
				case err, ok := <-watcher.Errors:
					if !ok {

					}
					log.Println("error: ", err)
			}
		}
	}()

	// Watch our web directory
	err = watcher.Add(WEBDIR)
	if err != nil {
		log.Fatal(err)
	}

	// Notify operator what we're doing
	fmt.Printf("Watching %s ...\n", WEBDIR)

	// Block so we can do this forever
	<-make(chan struct{})
}


// Top level setup function for serving static files.
func setupWeb() http.ServeMux {
  mux := http.NewServeMux()
  mux.Handle("/", http.FileServer(http.Dir("./dist")))
  return *mux
}


func main() {
	godotenv.Load()

	// Do an initial build so that assets are ready to be served
	fmt.Println("// initial build...")
	ops, result := makeWeb()
	displayResults(ops, result)

	var mux http.ServeMux = setupWeb()

	cert := os.Getenv(ENV_CERT)
	key := os.Getenv(ENV_KEY)

	fmt.Println("// starting server, watcher...")
  go func() {
		fmt.Println("Listening at :443 ...")
		http.ListenAndServeTLS(":https", cert, key, &mux)
	}()

	watchWeb()
}
