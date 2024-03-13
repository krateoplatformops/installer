//go:build integration
// +build integration

package secrets

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestGet(t *testing.T) {
	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	cli, err := NewClient(rc)
	if err != nil {
		t.Fatal(err)
	}

	dat, err := cli.Namespace("krateo-system").
		GetData(context.TODO(), "vcluster-k8s-certs", "ca.crt")
	if err != nil {
		t.Fatal(err)
	}

	caCrt := base64.StdEncoding.EncodeToString(dat)
	fmt.Println(caCrt)

}

func newRestConfig() (*rest.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
}
