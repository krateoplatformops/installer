package dynamic

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/itchyny/gojq"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func Extract(ctx context.Context, obj *unstructured.Unstructured, filter string) (any, error) {
	query, err := gojq.Parse(filter)
	if err != nil {
		return nil, err
	}

	var rawJson interface{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &rawJson)
	if err != nil {
		return nil, err
	}

	enc := newEncoder(false, 0)

	iter := query.RunWithContext(ctx, rawJson)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, err
		}
		if err := enc.encode(v); err != nil {
			return nil, err
		}
	}

	buf := strings.NewReader(enc.w.String())

	var xxx any
	err = json.NewDecoder(buf).Decode(&xxx)
	return xxx, err
}
