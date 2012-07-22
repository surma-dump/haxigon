package main

import (
	"bufio"
	"flag"
	"net"
	"path/filepath"
	"log"
	"io"
	"os"
	"os/signal"
	"os/exec"
	"syscall"
)

const (
	PORT_START = 5000
)

var (
	helpFlag = flag.Bool("help", false, "Show this help")
	signalAddr = flag.String("signal", "localhost:34122", "Address to listen on for change signals")
	repoDir = flag.String("repo", "", "Path to the reposetories")
	binDir = flag.String("bin", "/tmp/haxigon", "Path to the bin dir")
)

var (
	runningApps = map[string]*App{}
)

func main() {
	flag.Parse()


	if *helpFlag || *repoDir == "" || *binDir == "" {
		flag.PrintDefaults()
		return
	}

	pathSetup()

	go sigTermHandler()

	addr, e := net.ResolveTCPAddr("tcp4", *signalAddr)
	if e != nil {
		log.Fatalf("Could not parse signal address \"%s\": %s", *signalAddr, e)
	}
	l, e := net.ListenTCP("tcp4", addr)
	if e != nil {
		log.Fatalf("Could not listen on \"%s\": %s", *signalAddr, e)
	}
	log.Printf("Waiting for connections...")
	for {
		c, e := l.Accept()
		if e != nil {
			continue
		}
		go handleClient(c)
	}
}

func pathSetup() {
	var e error
	*repoDir, e = filepath.Abs(*repoDir)
	if e != nil {
		log.Fatalf("Could not absolutifiy \"%s\": %s", *repoDir, e)
	}
	*binDir, e = filepath.Abs(*binDir)
	if e != nil {
		log.Fatalf("Could not absolutifiy \"%s\": %s", *binDir, e)
	}
	e = os.MkdirAll(*binDir, os.FileMode(0755))
	if e != nil {
		log.Fatalf("Could not create bin dir: %s", e)
	}
}

func sigTermHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	for _, app := range runningApps {
		log.Printf("Destroying %s", app.Name)
		app.Destroy()
	}
	os.Exit(0)
}

func handleClient(rwc io.ReadWriteCloser) {
	defer rwc.Close()
	log := log.New(io.MultiWriter(os.Stderr, rwc), "", log.LstdFlags)
	br := bufio.NewReader(rwc)
	for {
		line, _, e := br.ReadLine()
		if e != nil {
			if e != io.EOF {
				log.Printf("Failed to read from client: %s", e)
			}
			return
		}
		appname := string(line)
		if appname == "" {
			continue
		}
		app, ok := runningApps[appname]
		if ok {
			e := app.Destroy()
			if e != nil {
				log.Printf("Could not stop old app: %s", e)
				return
			}
			delete(runningApps, appname)
		}
		e = cloneRepo(appname)
		if e != nil {
			log.Printf("Could not clone app repository: %s", e)
			return
		}

		app, e = StartApp(appname, filepath.Join(*binDir, appname))
		if e != nil {
			log.Printf("Could not start app: %s", e)
			os.RemoveAll(filepath.Join(*binDir, appname))
			return
		}
		runningApps[appname] = app
		gen
	}
}

func cloneRepo(appname string) error {
	cmd := exec.Command("git", "clone", filepath.Join(*repoDir, appname), filepath.Join(*binDir, appname))
	return cmd.Run()
}

func findFreePort() int {
	for port := PORT_START; port < 65535; port++ {
		free := true
		for _, app := range runningApps {
			if app.Port == port {
				free = false
				break
			}
		}
		if free {
			return port
		}
	}
	panic("No free port found")
}
