package handler

import (
	"github.com/hwcer/cosnet/sockets"
	"github.com/hwcer/registry"
	"reflect"
)

type RegistryFilter func(s *registry.Service, pr, fn reflect.Value) bool
type RegistryCaller func(socket *sockets.Socket, msg sockets.Message, pr reflect.Value, fn reflect.Value) (interface{}, error)
type RegistryMethod func(socket *sockets.Socket, msg sockets.Message) interface{}
type RegistryHandle interface {
	Caller(socket *sockets.Socket, msg sockets.Message, fn reflect.Value) interface{}
}
