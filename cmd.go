package main

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/text/transform"
	"io"
	"net"
	"strings"
	"time"
)

var (
	commonPrefix         = []byte("gatling.mysim.")
	filterPrefix         = bytes.Join([][]byte{commonPrefix, []byte("users.")}, nil)
	copyMeasurementPath  = bytes.Join([][]byte{commonPrefix, []byte("users.allUsers.active")}, nil)
	copyMeasurementLabel = []byte("allActiveUsers")
	previousTimestamp    = []byte("0000000000")
)

type filterTransformer struct {
	transform.NopResetter

	allActiveUsers []byte
}

func ExtractPathValues(line []byte) (path, value, timestamp []byte) {
	if len(line) <= 1 {
		return
	}

	parts := bytes.Split(line, []byte{' '})
	path = parts[0]
	value = parts[1]
	timestamp = parts[2]
	return
}

func (t filterTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	var removedBytes = 0
	origLines := bytes.Split(src, []byte{'\n'})
	lines := [][]byte{}
	for _, line := range origLines {
		if !bytes.HasPrefix(line, filterPrefix) {
			_, _, timeStamp := ExtractPathValues(line)

			if len(timeStamp) > 2 && !bytes.Equal(timeStamp, previousTimestamp) {
				copy(previousTimestamp, timeStamp)
				lines = append(lines, bytes.Join([][]byte{
					commonPrefix,
					copyMeasurementLabel,
					[]byte{' '},
					t.allActiveUsers, //value
					[]byte{' '},
					timeStamp, //timestamp
				}, nil))
			}

			lines = append(lines, line)
		} else {
			removedBytes += len(line) + 1
			value := bytes.Split(bytes.TrimPrefix(line, filterPrefix), []byte{' '})[1]

			if bytes.HasPrefix(line, copyMeasurementPath) {
				t.allActiveUsers = value
			}
		}
	}
	nSrc = len(src)
	nDst = copy(dst, bytes.Join(lines, []byte{'\n'}))
	if nDst < (nSrc - removedBytes) {
		err = errors.New("transform: short destination buffer")
	}
	return nDst, nSrc, err
}

func readUntilClosed(reader io.Reader) error {
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

	inConnTrans := transform.NewReader(inConn, filterTransformer{})

	inConnReader := io.TeeReader(inConnTrans, outConn)
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
