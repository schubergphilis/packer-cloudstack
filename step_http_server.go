package cloudstack

import (
	"fmt"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
)

// This step creates and runs the HTTP server that is serving the files
// specified by the 'http_files` configuration parameter in the template.
//
// Uses:
//   config *config
//   ui     packer.Ui
//
// Produces:
//   http_port int - The port the HTTP server started on.
type stepHTTPServer struct {
	l net.Listener
}

func (s *stepHTTPServer) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(config)
	ui := state.Get("ui").(packer.Ui)

	var httpPort uint = 0
	if config.HTTPDir == "" {
		// Save some dummy data to enable templating of
		// userdata work even if we are not running a web
		// server.
		state.Put("http_ip", "0.0.0.0")
		state.Put("http_port", "0")
		return multistep.ActionContinue
	}

	// Find an IP address for out HTTP server
	url, err := url.Parse(config.APIURL)
	if err != nil {
                state.Put("error", err)
                ui.Error(err.Error())
		return multistep.ActionHalt
	}
	conn, err := net.Dial("tcp", url.Host)
	if err != nil {
                state.Put("error", err)
                ui.Error(err.Error())
		return multistep.ActionHalt
	}
	httpIP, _, _ := net.SplitHostPort(conn.LocalAddr().String())
	conn.Close()

	// Find an available TCP port for our HTTP server
	var httpAddr string
	portRange := int(config.HTTPPortMax - config.HTTPPortMin)
	for {
		var err error
		var offset uint = 0

		if portRange > 0 {
			// Intn will panic if portRange == 0, so we
			// calculate an offset.
			offset = uint(rand.Intn(portRange))
		}

		httpPort = offset + config.HTTPPortMin
		httpAddr = fmt.Sprintf("%s:%d", httpIP, httpPort)
		log.Printf("Trying %v", httpAddr)
		s.l, err = net.Listen("tcp", httpAddr)
		if err == nil {
			break
		}
	}

	ui.Say(fmt.Sprintf("Starting HTTP server on %v", httpAddr))

	// Start the HTTP server and run it in the background
	fileServer := http.FileServer(http.Dir(config.HTTPDir))
	server := &http.Server{Addr: httpAddr, Handler: fileServer}
	go server.Serve(s.l)

	// Save the address into the state so it can be accessed in the future
	state.Put("http_ip", httpIP)
	state.Put("http_port", fmt.Sprintf("%d", httpPort))

	return multistep.ActionContinue
}

func (s *stepHTTPServer) Cleanup(multistep.StateBag) {
	if s.l != nil {
		// Close the listener so that the HTTP server stops
		s.l.Close()
	}
}
