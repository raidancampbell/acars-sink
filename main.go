package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raidancampbell/acars-sink/internal/decoder"
	"github.com/raidancampbell/acars-sink/internal/server"
	"github.com/raidancampbell/acars-sink/internal/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	udpAddr = ":5555"
	tcpAddr = ":5555"
	vdlm2Addr = ":5556"
	dbPath  = "./acars.db"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	udpListener := server.NewListener(udpAddr)
	tcpListener := server.NewTCPListener(tcpAddr)
	vdlm2Listener := server.NewTCPListener(vdlm2Addr)
	store := storage.NewStore(dbPath)

	if err := store.Init(); err != nil {
		log.Fatal().Err(err).Msg("storage init failed")
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Error().Err(err).Msg("storage close failed")
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	handlePayload := func(payload []byte, remote string, source string) error {
		receivedAt := time.Now()
		var msg decoder.Message
		if err := json.Unmarshal(payload, &msg); err != nil {
			log.Warn().Err(err).Str("remote", remote).Str("source", source).Msg("invalid JSON")
			return nil
		}

		event := log.Info().
			Str("remote", remote).
			Str("source", source).
			Str("aircraft", msg.Aircraft).
			Str("flight", msg.Flight).
			Str("message_type", msg.Type).
			Str("station", msg.Station).
			Str("label", msg.Label).
			Str("channel", string(msg.Channel)).
			Str("registration", msg.Registration).
			Str("icao", msg.ICAO)

		if err := store.Insert(ctx, receivedAt, source, string(payload), msg); err != nil {
			log.Error().Err(err).
				Str("remote", remote).
				Str("source", source).
				Str("aircraft", msg.Aircraft).
				Str("flight", msg.Flight).
				Str("message_type", msg.Type).
				Str("station", msg.Station).
				Msg("db insert failed")
			return nil
		}

		event.Msg("message stored")
		return nil
	}

	errCh := make(chan error, 2)

	log.Info().Str("addr", udpAddr).Msg("listening for UDP")
	go func() {
		errCh <- udpListener.Start(ctx, func(payload []byte, addr *net.UDPAddr) error {
			return handlePayload(payload, addr.String(), "acars")
		})
	}()

	log.Info().Str("addr", tcpAddr).Msg("listening for TCP")
	go func() {
		errCh <- tcpListener.Start(ctx, func(payload []byte, addr net.Addr) error {
			return handlePayload(payload, addr.String(), "acars")
		})
	}()

	log.Info().Str("addr", vdlm2Addr).Msg("listening for VDLM2 TCP")
	go func() {
		errCh <- vdlm2Listener.Start(ctx, func(payload []byte, addr net.Addr) error {
			return handlePayload(payload, addr.String(), "vdlm2")
		})
	}()

	if err := <-errCh; err != nil && ctx.Err() == nil {
		log.Fatal().Err(err).Msg("listener stopped")
	}
}
