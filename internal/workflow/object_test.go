package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/krateoplatformops/installer/apis/releases/v1alpha1"
	"github.com/krateoplatformops/installer/internal/cache"
)

func TestObject(t *testing.T) {
	dat, err := loadSample("obj-sample.json")
	if err != nil {
		t.Fatal(err)
	}

	res := v1alpha1.Object{}
	err = json.Unmarshal(dat, &res)
	if err != nil {
		t.Fatal(err)
	}

	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	env := cache.New[string, string]()
	env.Set("KUBECONFIG_CAKEY", "XXXXX")

	oh := &objHandler{
		rc:  rc,
		env: env,
		ns:  "krateo-system",
	}

	err = oh.Do(context.TODO(), &res)
	if err != nil {
		t.Fatal(err)
	}

}
