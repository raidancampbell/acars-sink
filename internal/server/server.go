package server

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Handler receives a raw UDP payload.
type Handler func(payload []byte, addr *net.UDPAddr) error

// TCPHandler receives a raw TCP payload.
type TCPHandler func(payload []byte, addr net.Addr) error

// Listener will encapsulate UDP socket handling.
type Listener struct {
	Addr string
}

func NewListener(addr string) *Listener {
	return &Listener{Addr: addr}
}

// TCPListener encapsulates TCP socket handling.
type TCPListener struct {
	Addr string
}

func NewTCPListener(addr string) *TCPListener {
	return &TCPListener{Addr: addr}
}

// Start listens for UDP packets and forwards raw JSON to the handler.
func (l *Listener) Start(ctx context.Context, handler Handler) error {
	udpAddr, err := net.ResolveUDPAddr("udp", l.Addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf := make([]byte, 65535)

	for {
		if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			return err
		}

		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					continue
				}
			}
			return err
		}

		log.Debug().Str("remote", addr.String()).Int("bytes", n).Msg("udp packet received")

		payload := make([]byte, n)
		copy(payload, buf[:n])

		if handlerErr := handler(payload, addr); handlerErr != nil {
			return handlerErr
		}
	}
}

// Start listens for TCP connections and forwards newline-delimited JSON to the handler.
func (l *TCPListener) Start(ctx context.Context, handler TCPHandler) error {
	ln, err := net.Listen("tcp", l.Addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	const tcpScanTimeout = 2 * time.Minute

	for {
		if tcpLn, ok := ln.(*net.TCPListener); ok {
			if err := tcpLn.SetDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
				return err
			}
		}

		conn, err := ln.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					continue
				}
			}
			return err
		}

		log.Debug().Str("remote", conn.RemoteAddr().String()).Msg("tcp connection accepted")

		go func(c net.Conn) {
			defer c.Close()
			done := make(chan struct{})
			go func() {
				select {
				case <-ctx.Done():
					_ = c.Close()
				case <-done:
				}
			}()
			defer close(done)

			reader := bufio.NewReaderSize(c, 1024*1024)

			for {
				if err := c.SetReadDeadline(time.Now().Add(tcpScanTimeout)); err != nil {
					log.Warn().Err(err).Str("remote", c.RemoteAddr().String()).Msg("tcp set read deadline failed")
					return
				}

				line, err := reader.ReadString('\n')
				if err != nil {
					if errors.Is(err, io.EOF) {
						log.Info().Str("remote", c.RemoteAddr().String()).Msg("tcp connection closed")
						return
					}
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						log.Info().Str("remote", c.RemoteAddr().String()).Dur("timeout", tcpScanTimeout).Msg("tcp scan timeout")
						continue
					}
					log.Warn().Err(err).Str("remote", c.RemoteAddr().String()).Msg("tcp read failed")
					return
				}

				line = strings.TrimRight(line, "\r\n")
				if line == "" {
					continue
				}

				payload := []byte(line)
				log.Debug().Str("remote", c.RemoteAddr().String()).Int("bytes", len(payload)).Msg("tcp line received")
				if handlerErr := handler(payload, c.RemoteAddr()); handlerErr != nil {
					return
				}
			}
		}(conn)
	}
}
