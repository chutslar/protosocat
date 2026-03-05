package ws

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
)

type WSServer struct {
	Port     int
	BasePath string
	Send     chan []byte
	Receive  chan []byte
	Errors   chan error
	Info     chan string
}

func (s *WSServer) HandleWebsocket() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			s.Errors <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		s.Info <- "Client connected"

		go func() {
			for {
				_, data, err := conn.Read(context.Background())
				if err != nil {
					s.Errors <- err
					return
				}
				s.Receive <- data
			}
		}()

		for data := range s.Send {
			err := conn.Write(context.Background(), websocket.MessageBinary, data)
			if err != nil {
				s.Errors <- err
				return
			}
		}
	}
}

func (s *WSServer) Run() {
	path := "/"
	if s.BasePath != "" {
		path = s.BasePath
	}
	go func() {
		http.HandleFunc(path, s.HandleWebsocket())

		s.Info <- fmt.Sprintf("Server running on port %d", s.Port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil)
		if !errors.Is(err, http.ErrServerClosed) {
			s.Errors <- fmt.Errorf("server error: %w", err)
		}
	}()
}
