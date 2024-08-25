package main

import (
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	var source string
	var destination string

	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "-p" {
			source = ":" + strings.Split(os.Args[i+1], ":")[1]
			destination = ":" + strings.Split(os.Args[i+1], ":")[0]
		}
	}

	server, err := net.Listen("tcp", destination)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", destination, err)
	}
	defer server.Close()

	func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				continue
			}
			go func() {
				localCall, err := net.Dial("tcp", source)
				if err != nil {
					log.Printf("Failed to connect to localCall: %v", err)
					return
				}
				go io.Copy(localCall, conn)
				io.Copy(conn, localCall)
			}()
		}
	}()
}
