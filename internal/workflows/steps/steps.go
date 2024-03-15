package steps

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

type Handler interface {
	Namespace(ns string)
	Handle(ctx context.Context, id string, in *runtime.RawExtension) error
}
