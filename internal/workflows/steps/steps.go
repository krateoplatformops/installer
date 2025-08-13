package steps

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

type Op int

const (
	Create Op = iota + 1
	Update
	Delete
)

type Handler[T any] interface {
	Namespace(ns string)
	Op(op Op)
	Handle(ctx context.Context, id string, in *runtime.RawExtension) (T, error)
}
