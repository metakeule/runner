package runner

import (
	"fmt"
	//	"koelnart/lib/pid"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

func dispatcher(rw http.ResponseWriter, req *http.Request) {
	a := strings.Split(req.Host, ":")
	host := a[0]
	projectName, foundVhost := vhosts[host]
	if !foundVhost {
		log.Printf("WARN: unknown host: %s\n", req.Host)
		return
	}
	project, foundProject := projects[projectName]
	if !foundProject {
		log.Printf("WARN: unknown project: %s\n", projectName)
		return
	}
	project.ServeHTTP(rw, req)
}

type Project struct {
	Name   string
	Mount  func(*http.ServeMux)
	Vhosts []string
	OnExit func()
}

func (ø Project) Check() error {
	if ø.Name == "" {
		return fmt.Errorf("no name given")
	}
	if ø.Mount == nil {
		return fmt.Errorf("no mount function set")
	}
	if len(ø.Vhosts) == 0 {
		return fmt.Errorf("no vhosts set")
	}
	return nil
}

var (
	onExits = []func(){}

	// overwrite these values to change them before calling Serve()
	MainFile       = "run/main.go"
	NumCPU         = runtime.NumCPU()
	Port           = 8080
	Host           = "localhost"
	MaxHeaderBytes = 1 << 20          // maximum size of headers (1 << 10 == 1KB, 1 << 20 == 1 MB, 1 << 30 == 1 GB)
	ReadTimeout    = time.Minute * 2  // maximum duration before timing out read of the request
	WriteTimeout   = time.Second * 90 // maximum duration before timing out write of the response

	// map projects to muxer
	projects = map[string]*http.ServeMux{}

	// map hosts to projects
	vhosts = map[string]string{}
)

func Add(project Project) {
	err := project.Check()
	if err != nil {
		panic(err.Error())
	}
	if _, foundProject := projects[project.Name]; foundProject {
		panic("project " + project.Name + " is already registered")
	}
	m := http.NewServeMux()
	projects[project.Name] = m
	project.Mount(m)
	for _, vh := range project.Vhosts {
		if pr, foundVhost := vhosts[vh]; foundVhost {
			panic("vhost " + vh + " is already registered for project " + pr)
		}
		vhosts[vh] = project.Name
	}
	if project.OnExit != nil {
		onExits = append(onExits, project.OnExit)
	}
}

func KillOld(pidPath string) {
	kill(pidPath)
	write(pidPath)
	onExits = append(onExits, func() { os.Remove(pidPath) })
}

func Serve() {
	handleInterrupt(func() {
		for _, e := range onExits {
			e()
		}
	})
	runtime.GOMAXPROCS(NumCPU)
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%v", Host, Port),
		Handler:        http.HandlerFunc(dispatcher),
		ReadTimeout:    ReadTimeout,    // maximum duration before timing out read of the request
		WriteTimeout:   WriteTimeout,   // maximum duration before timing out write of the response
		MaxHeaderBytes: MaxHeaderBytes, // max size of headers
	}

	log.Printf("server (%v cores) is listening on http://%s:%v\n", NumCPU, Host, Port)
	err := server.ListenAndServe()
	if err != nil {
		log.Printf("could not start server: %s\n", err.Error())
		os.Exit(1)
	}
}
