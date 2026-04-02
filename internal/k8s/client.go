package k8s

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Config holds configuration for the Kubernetes client
type Config struct {
	KubeconfigPath string
	QPS            float32
	Burst          int
	Timeout        time.Duration
	UserAgent      string
}

// Client wraps Kubernetes client-go components
type Client struct {
	clientset       kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	restMapper      meta.RESTMapper
	restConfig      *rest.Config
}

// NewClient creates a new Kubernetes client from the given config
func NewClient(cfg Config) (*Client, error) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", cfg.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig: %w", err)
	}

	restConfig.QPS = cfg.QPS
	restConfig.Burst = cfg.Burst
	restConfig.Timeout = cfg.Timeout
	if cfg.UserAgent != "" {
		restConfig.UserAgent = cfg.UserAgent
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	discoveryClient := clientset.Discovery()

	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return nil, fmt.Errorf("getting API group resources: %w", err)
	}

	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	return &Client{
		clientset:       clientset,
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
		restMapper:      mapper,
		restConfig:      restConfig,
	}, nil
}

// RefreshDiscovery refreshes the REST mapper with updated API group resources
func (c *Client) RefreshDiscovery() error {
	groupResources, err := restmapper.GetAPIGroupResources(c.discoveryClient)
	if err != nil {
		return fmt.Errorf("refreshing API group resources: %w", err)
	}
	c.restMapper = restmapper.NewDiscoveryRESTMapper(groupResources)
	return nil
}

// TestConnection validates connectivity to the Kubernetes cluster
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("cannot connect to Kubernetes cluster: %w", err)
	}
	return nil
}

// Clientset returns the underlying kubernetes.Interface
func (c *Client) Clientset() kubernetes.Interface {
	return c.clientset
}

// DynamicClient returns the underlying dynamic.Interface
func (c *Client) DynamicClient() dynamic.Interface {
	return c.dynamicClient
}

// RESTConfig returns the underlying rest config
func (c *Client) RESTConfig() *rest.Config {
	return c.restConfig
}
