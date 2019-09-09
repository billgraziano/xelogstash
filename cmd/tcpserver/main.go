package main

/*
	A simple TCP server that listens and pretends to be logstash.
	Not perfect but good enough.
*/

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"strconv"
)

func handleConn(c net.Conn, number int, maxPrint int) {
	defer c.Close()

	var msgNum int
	//log.Printf("New conn: %v", c)
	rw := bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))
	for {
		msg, err := rw.ReadString('\n')
		msgNum++
		switch {
		case err == io.EOF:
			log.Println("Reached EOF - close this connection.\n   ---")
			return
		case err != nil:
			//log.Println(c.)
			if op, ok := err.(*net.OpError); ok {
				log.Printf("connection closed (*net.OpError: %s)\r\n", op.Op)
				return
			}
			log.Println("Error reading command:", err)
			log.Printf("%T %+v", err, err)
			return
		}

		var printSize int
		if len(msg) > maxPrint+1 {
			printSize = maxPrint
		} else {
			printSize = len(msg) - 1
		}
		if printSize < 1 {
			printSize = 0
		}

		log.Printf("*** conn #%d (%d) - bytes: %d", number, msgNum, len(msg))
		// fmt.Println("--------------------------------------------------------")
		// fmt.Printf("%s\r\n", msg[0:printSize])
		// fmt.Println("--------------------------------------------------------")
	}
}

func main() {

	listen := ":8888"
	maxPrint := 90

	if len(os.Args) >= 2 {
		listen = os.Args[1]
	}
	if len(os.Args) == 3 {
		maxPrint, _ = strconv.Atoi(os.Args[2])
	}

	var connCount int

	l, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	log.Println("Listening on", listen)

	for {
		// accept connection
		log.Println("Waiting for new connection...")
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		connCount++
		log.Println("Connection:", connCount)

		// handle connection
		go handleConn(conn, connCount, maxPrint)
	}
}
