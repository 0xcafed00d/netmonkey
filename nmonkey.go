package main

import (
	"fmt"
	"io"
	"log"
	"os"
)

func docopy(r io.Reader, w io.Writer, errchan chan (error)) {
	buffer := make([]byte, 1024)

	for {
		nr, err := r.Read(buffer)
		if err != nil {
			errchan <- err
			return
		}

		nw, err := w.Write(buffer[0:nr])
		if err != nil {
			errchan <- err
			return
		}

		if nw != nr {
			errchan <- io.ErrShortWrite
			return
		}
	}
}

func CreateConnecton(c ConnectInfo, errch chan error) {

	log.Printf("Making Connection [%s -> %s]", c.From, c.To)

	var source io.Reader = GetEndPoint(c.From)
	if source == nil {
		errch <- fmt.Errorf("unknown Endpoint name for conenction source: %v", c.From)
	}

	for _, finfo := range c.Filters {
		if maker, ok := FilterFactory[finfo.Name]; ok {
			filter, err := maker(finfo.Config)
			if err != nil {
				errch <- err
				return
			}
			ExitOnError(err, 1)
			filter.SetSource(source)
			source = filter
		} else {
			errch <- fmt.Errorf("unknown Filter Type: %v", finfo.Name)
			return
		}
	}

	dest := GetEndPoint(c.To)
	if dest == nil {
		errch <- fmt.Errorf("unknown Endpoint name for connection destination: %v", c.To)
		return
	}

	log.Printf("Connection made: [%s -> %s]", c.From, c.To)

	docopy(source, dest, errch)
}

type RequestEndPoint struct {
	name   string
	epchan chan EndPoint
}

var epRequestChan = make(chan RequestEndPoint)

func GetEndPoint(name string) EndPoint {
	epchan := make(chan EndPoint)
	log.Printf("Requesting endpoint: %s", name)
	epRequestChan <- RequestEndPoint{name, epchan}
	return <-epchan
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Invalid Args: usage: netmonkey <config file>")
		os.Exit(1)
	}

	errch := make(chan error)
	epch := make(chan EndPoint)
	endPoints := make(map[string]EndPoint)
	var requests []RequestEndPoint

	config, err := ReadConfig(os.Args[1])
	ExitOnError(err, 1)

	err = CreateEndPoints(config.EndPoints, "", epch, errch)
	ExitOnError(err, 1)

	for _, c := range config.Connections {
		go CreateConnecton(c, errch)
	}

	for {
		select {
		case epreq := <-epRequestChan:
			requests = append(requests, epreq)

		case ep := <-epch:
			log.Printf("Endpoint %s created.", ep.Name())
			endPoints[ep.Name()] = ep
			// if any endpoints are dependent on this one create them
			CreateEndPoints(config.EndPoints, ep.Name(), epch, errch)

		case err := <-errch:
			for _, ep := range endPoints {
				ep.Close()
			}
			ExitOnError(err, 1)
		}

		if len(requests) > 0 {
			var unsatisfied []RequestEndPoint
			// iterate through the outsating requests and see if we can satisfy any of them
			for _, req := range requests {
				if ep, ok := endPoints[req.name]; ok {
					req.epchan <- ep
				} else {
					unsatisfied = append(unsatisfied, req)
				}
			}
			requests = unsatisfied
		}
	}
}
