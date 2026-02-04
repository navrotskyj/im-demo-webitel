package grpc_client

import (
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client[T any] struct {
	conn *grpc.ClientConn
	Api  T
}

var conns sync.Map

type config struct {
	dialOptions []grpc.DialOption
	lbPolicy    string
}

type Option func(*config)

func WithGrpcOptions(opts ...grpc.DialOption) Option {
	return func(c *config) {
		c.dialOptions = append(c.dialOptions, opts...)
	}
}

func WithLBPolicy(policy string) Option {
	return func(c *config) {
		c.lbPolicy = policy
	}
}

func NewClient[T any](consulTarget, service string, api func(grpc.ClientConnInterface) T, opts ...Option) (*Client[T], error) {

	cfg := &config{
		lbPolicy:    "wbt_round_robin",
		dialOptions: []grpc.DialOption{
			//grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if len(cfg.dialOptions) == 0 {
		cfg.dialOptions = append(cfg.dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	dsn := fmt.Sprintf("wbt://%s/%s?wait=15s", consulTarget, service)

	dialOpts := append(cfg.dialOptions,
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy": "%s"}`, cfg.lbPolicy)),
	)

	actual, _ := conns.LoadOrStore(dsn, func() interface{} {
		conn, err := grpc.NewClient(dsn, dialOpts...)
		if err != nil {
			return err // Тут треба бути обережним, краще зберігати структуру-обгортку
		}
		return conn
	}())

	conn, ok := actual.(*grpc.ClientConn)
	if !ok {
		return nil, fmt.Errorf("failed to create or retrieve connection")
	}

	return &Client[T]{
		conn: conn,
		Api:  api(conn),
	}, nil
}

func (c *Client[T]) Close() error {
	return c.conn.Close()
}

func (c *Client[T]) Start() error {
	return nil
}
