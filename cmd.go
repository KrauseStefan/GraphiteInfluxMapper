package main

import (
	"fmt"
	"net"
	"strings"
	"time"
	"io"
)

func readUntilClosed(reader io.Reader) error{
	bytesToRead := 4096
	data := make([]byte, bytesToRead, bytesToRead)

	for {
		readBytes, err := reader.Read(data)
		if err != nil {
			return err
		}
		if readBytes > 0 {
			dataStrs := strings.Split(string(data[:readBytes]), "\n")
			dataStr := strings.Join(dataStrs, "\\n\n")
			fmt.Printf("%s\n\n", dataStr)
		} else {
			fmt.Print("*")
			time.Sleep(1 * time.Second)
		}
	}
}

func handleConnection(inConn net.Conn) error {
	fmt.Println("New Incomming connection")

	outConn, err := net.Dial("tcp", "localhost:2008")
	if err != nil {
		return err
	}
	fmt.Printf("Proxying to %s\n", outConn.RemoteAddr())

	inConnReader := io.TeeReader(inConn, outConn)
	outConnReader := io.TeeReader(outConn, inConn)

	go (func() {
		err := readUntilClosed(outConnReader)
		if err != nil {
			fmt.Println(err)
		}
	})()

	err = readUntilClosed(inConnReader)

	outConn.Close()

	return err
}

func main() {
	protocol, listenAddress := "tcp", ":2003"
	ln, err := net.Listen(protocol, listenAddress)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Server started: %s %s\n", ln.Addr().Network(), ln.Addr())

	conn, err := ln.Accept()
	if err != nil {
		fmt.Println(err)
	}

	err = handleConnection(conn)
	if err != nil {
		fmt.Println(err)
	}

	err = ln.Close()
	if err != nil {
		fmt.Println(err)
	}

	// To allow connection to be closed in the other direction as well
	time.Sleep(2 * time.Second)
}
