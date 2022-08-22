package handler

import (
	"github.com/hwcer/registry"
)

type RegistryFilter func(s *registry.Service, node *registry.Node) bool
type RegistryCaller func(ctx *Context, node *registry.Node) (interface{}, error)

type RegistryHandle interface {
	Caller(ctx *Context, node *registry.Node) interface{}
}
