package secrets

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

const (
	resourceName = "secrets"
)

func NewClient(rc *rest.Config) (*Client, error) {
	gv := schema.GroupVersion{
		Group:   "",
		Version: "v1",
	}

	sb := runtime.NewSchemeBuilder(
		func(reg *runtime.Scheme) error {
			reg.AddKnownTypes(
				gv,
				&corev1.Secret{},
				&corev1.SecretList{},
				&metav1.ListOptions{},
				&metav1.GetOptions{},
				&metav1.DeleteOptions{},
				&metav1.CreateOptions{},
				&metav1.UpdateOptions{},
				&metav1.PatchOptions{},
			)
			return nil
		})

	s := runtime.NewScheme()
	sb.AddToScheme(s)

	config := *rc
	config.APIPath = "/api"
	config.GroupVersion = &gv
	config.NegotiatedSerializer = serializer.NewCodecFactory(s).
		WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	cli, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	pc := runtime.NewParameterCodec(s)

	return &Client{rc: cli, pc: pc}, nil
}

type Client struct {
	rc rest.Interface
	pc runtime.ParameterCodec
	ns string
}

func (c *Client) Namespace(ns string) *Client {
	c.ns = ns
	return c
}

func (c *Client) Get(ctx context.Context, name string) (result *corev1.Secret, err error) {
	result = &corev1.Secret{}
	err = c.rc.Get().
		Namespace(c.ns).
		Resource(resourceName).
		Name(name).
		//VersionedParams(&options, c.pc).
		Do(ctx).
		Into(result)
	return
}

func (c *Client) GetData(ctx context.Context, name, key string) ([]byte, error) {
	res, err := c.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	val, ok := res.Data[key]
	if !ok {
		return nil, fmt.Errorf("ca.crt value not found in secret %s/%s",
			res.GetNamespace(), res.GetName())
	}
	return val, nil
}
