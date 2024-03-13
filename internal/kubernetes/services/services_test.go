//go:build integration
// +build integration

package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestServiceGetIP(t *testing.T) {
	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	cli, err := NewClient(rc)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := cli.Namespace("krateo-system").
		GetIP(context.TODO(), "vcluster-k8s")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(addr)
}

func newRestConfig() (*rest.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
}
