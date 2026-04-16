package registry

import (
	"time"

	kretcd "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdConfig struct {
	Endpoints   []string
	DialTimeout time.Duration
}

func NewEtcdRegistrar(cfg EtcdConfig) (*kretcd.Registry, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	})
	if err != nil {
		return nil, err
	}

	return kretcd.New(client), nil
}
