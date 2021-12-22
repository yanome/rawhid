package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/gousb"
)

const VID = 0x16C0
const PID = 0x0486

// returns IN and OUT endpoints
func getEndpoints(intf *gousb.Interface) (in, out gousb.EndpointDesc) {
	for _, epDesc := range intf.Setting.Endpoints {
		switch epDesc.Direction {
		case gousb.EndpointDirectionIn:
			in = epDesc
		case gousb.EndpointDirectionOut:
			out = epDesc
		}
	}
	return
}

// reads a line from STDIN and writes first byte to USB device
func write(out *gousb.OutEndpoint) {
	for { // this loop is to catch an EOL (CTRL+D) on STDIN
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			s := scanner.Bytes()
			if len(s) > 0 {
				log.Printf("Sending byte %#02x %#08b", s[:1], s[:1])
				n, err := out.Write(s[:1])
				if err != nil || n != 1 {
					log.Fatalf("Error, %v bytes were sent: %v", n, err)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("IO error: %v", err)
		}
	}
}

// reads from USB device
func read(in *gousb.InEndpoint, size int) {
	buffer := make([]byte, size)
	for {
		n, err := in.Read(buffer)
		if err != nil {
			log.Fatalf("Error, %v bytes were read: %v", n, err)
		}
		if n > 0 {
			log.Printf("Read %v bytes", n)
			print(buffer, n)
		}
	}
}

// prints formatted data
func print(data []byte, size int) {
	var b strings.Builder
	for s, e := 0, 0; s < size; s = e {
		e += 16
		if e > size {
			e = size
		}
		b.Reset()
		for _, d := range data[s:e] {
			fmt.Fprintf(&b, "%02x ", d)
		}
		log.Print(b.String())
	}
}

func main() {
	// create context
	ctx := gousb.NewContext()
	// deferred context close
	defer ctx.Close()

	// open device
	dev, err := ctx.OpenDeviceWithVIDPID(VID, PID)
	if err != nil {
		log.Fatalf("Error listing devices: %v", err)
	}
	if dev == nil {
		log.Fatalf("Device %04x:%04x not found", VID, PID)
	}
	// deferred device close
	defer dev.Close()

	// detach kernel driver
	dev.SetAutoDetach(true)

	// claim default interface
	intf, done, err := dev.DefaultInterface()
	if err != nil {
		log.Fatalf("Error claiming interface: %v", err)
	}
	// deferred interface release
	defer done()

	in, out := getEndpoints(intf)

	// write (on a goroutine)
	outE, err := intf.OutEndpoint(out.Number)
	if err != nil {
		log.Fatalf("Error preparing OUT endpoint (%v): %v", out.Address, err)
	}
	go write(outE)

	// read
	inE, err := intf.InEndpoint(in.Number)
	if err != nil {
		log.Fatalf("Error preparing IN endpoint (%v): %v", in.Address, err)
	}
	read(inE, in.MaxPacketSize)
}
