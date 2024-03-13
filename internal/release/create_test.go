package release

import (
	"context"
	"testing"
)

func TestCreate(t *testing.T) {
	cr, err := decodeSample()
	if err != nil {
		t.Fatal(err)
	}

	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	hc, err := helmClientForNamespace(rc, cr.GetNamespace())
	if err != nil {
		t.Fatal(err)
	}

	err = Create(context.TODO(), hc, cr)
	if err != nil {
		t.Fatal(err)
	}
}
