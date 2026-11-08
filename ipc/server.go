package ipc

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

// Dispatcher handles IPC commands by the running application.
type Dispatcher interface {
	HandleIPC(req Request) Response
}

// Server listens on a Unix domain socket (a named pipe on Windows) for IPC
// commands.
type Server struct {
	listener net.Listener
	sockPath string
	disp     Dispatcher
	done     chan struct{}
	wg       sync.WaitGroup
	cleanup  func()
}

// NewServer creates and starts an IPC server on the given socket path (a
// named pipe path on Windows).
func NewServer(sockPath string, disp Dispatcher) (*Server, error) {
	ln, cleanup, err := listen(sockPath)
	if err != nil {
		return nil, err
	}

	s := &Server{
		listener: ln,
		sockPath: sockPath,
		disp:     disp,
		done:     make(chan struct{}),
		cleanup:  cleanup,
	}
	s.wg.Add(1)
	go s.acceptLoop()

	log.Printf("IPC server listening on %s", sockPath)
	return s, nil
}

// Close shuts down the IPC server and cleans up socket files.
func (s *Server) Close() error {
	close(s.done)
	err := s.listener.Close()
	s.wg.Wait()
	if s.cleanup != nil {
		s.cleanup()
	}
	return err
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			writeResponse(conn, Response{OK: false, Error: "invalid JSON: " + err.Error()})
			continue
		}

		resp := s.disp.HandleIPC(req)
		writeResponse(conn, resp)
	}
}

func writeResponse(conn net.Conn, resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		data = []byte(`{"ok":false,"error":"marshal error"}`)
	}
	data = append(data, '\n')
	conn.Write(data)
}
