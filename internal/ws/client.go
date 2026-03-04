package ws

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coder/websocket"
)

type WSClient struct {
	URL     string
	Send    chan []byte
	Receive chan []byte
	Error   chan error
	Info    chan string
}

func (c *WSClient) runWithDelay(delay time.Duration) {
	if delay.Seconds() > 0 {
		time.Sleep(delay)
	}

	conn, _, err := websocket.Dial(
		context.Background(),
		c.URL,
		&websocket.DialOptions{},
	)
	if err != nil {
		if delay == 0 {
			delay = 10 * time.Second
		} else {
			delay = min(delay*2, 2*time.Minute)
		}
		log.Println(err)
		c.Error <- fmt.Errorf("failed to connect to %s,\nretrying in %s", c.URL, delay.String())
		go c.runWithDelay(delay)
		return
	}

	c.Info <- fmt.Sprintf("Connected to %s", c.URL)

	go func() {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				c.Error <- err
				return
			}
			c.Receive <- data
		}
	}()

	go func() {
		for data := range c.Send {
			if err := conn.Write(context.Background(), websocket.MessageBinary, data); err != nil {
				c.Error <- err
				return
			}
		}
	}()
}

func (c *WSClient) Run() {
	go c.runWithDelay(0)
}
