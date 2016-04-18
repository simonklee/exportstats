package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/simonz05/exportstats"
	"github.com/simonz05/util/ioutil"
	"github.com/simonz05/util/log"
	"github.com/simonz05/util/sig"
)

var (
	help           = flag.Bool("h", false, "show help text")
	laddr          = flag.String("http", ":6070", "set bind address for the HTTP server")
	accessToken    = flag.String("token", "", "stathat accesstoken")
	version        = flag.Bool("version", false, "show version number and exit")
	configFilename = flag.String("config", "config.toml", "config file path")
	cpuprofile     = flag.String("debug.cpuprofile", "", "write cpu profile to file")
)

var Version = "0.1.0"

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	log.Println("start exportstats service …")

	if *version {
		fmt.Fprintln(os.Stdout, Version)
		return
	}

	if *help {
		flag.Usage()
		os.Exit(1)
	}

	if *laddr == "" {
		log.Fatal("Listen address required")
	}

	if *accessToken == "" {
		log.Fatal("access token required")
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	_ = exportstats.NewServer(*accessToken)
	err := ListenAndServe(*laddr)

	if err != nil {
		log.Errorln(err)
	}
}

func ListenAndServe(laddr string) error {
	l, err := net.Listen("tcp", laddr)

	if err != nil {
		return err
	}

	log.Printf("Listen on %s", l.Addr())

	closer := ioutil.MultiCloser([]io.Closer{l})
	sig.TrapCloser(closer)
	err = http.Serve(l, nil)
	log.Printf("Shutting down ..")
	return err
}
