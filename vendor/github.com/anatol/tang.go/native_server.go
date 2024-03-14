package tang

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"testing"
)

// NativeServer is a server implementation that redirects requests to the native "tangd" binary.
// This code is useful for tests or when one needs a wrapper around tangd binary.
type NativeServer struct {
	KeysDir   string
	Port      int
	tangdPath string
	listener  net.Listener
}

// NewNativeServer creates instance of a native Tang server
func NewNativeServer(keysDir string, port int) (*NativeServer, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	// different OS use different tang server binary location
	tangLocations := []string{
		"/usr/lib/",
		"/usr/lib/x86_64-linux-gnu/",
	}

	var tangdPath string

	for _, l := range tangLocations {
		if _, err := os.Stat(l + "tangd"); err == nil {
			tangdPath = l + "tangd"
			break
		}
	}
	if tangdPath == "" {
		return nil, fmt.Errorf("unable to find tangd binary")
	}

	s := &NativeServer{
		KeysDir:   keysDir,
		Port:      l.Addr().(*net.TCPAddr).Port,
		tangdPath: tangdPath,
		listener:  l,
	}
	go s.Serve()
	return s, nil
}

// Stop stops the server
func (s *NativeServer) Stop() {
	_ = s.listener.Close()
}

// Serve serves HTTP requests
func (s *NativeServer) Serve() {
	for {
		conn, err := s.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			log.Println("accept error", err)
			return
		}
		s.handleConection(conn)
		if err := conn.Close(); err != nil {
			log.Print(err)
		}
	}
}

func (s *NativeServer) handleConection(conn net.Conn) {
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		log.Println("read error", err)
		return
	}
	if n == 0 {
		return
	}

	tangCmd := exec.Command(s.tangdPath, s.KeysDir)
	tangCmd.Stdin = bytes.NewReader(buf[:n])
	tangCmd.Stdout = conn
	if testing.Verbose() {
		tangCmd.Stderr = os.Stderr
	}
	if err := tangCmd.Run(); err != nil {
		log.Println(err)
	}
}
