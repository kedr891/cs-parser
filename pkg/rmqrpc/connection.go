package rmqrpc

import (
	"context"
	"errors"
)

type Config struct {
	URL string
}

type Connection struct{}

func New(consumerExchange string, cfg Config) *Connection {
	return &Connection{}
}

func (c *Connection) AttemptConnect() error {
	return errors.New("not implemented")
}

func (c *Connection) Start(ctx context.Context) error {
	return errors.New("not implemented")
}

func (c *Connection) Shutdown() error {
	return errors.New("not implemented")
}
