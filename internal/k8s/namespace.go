package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rs/zerolog/log"
)

// EnsureNamespace creates a namespace if it doesn't exist
func (c *Client) EnsureNamespace(ctx context.Context, name string, dryRun bool) error {
	if dryRun {
		log.Info().Msgf("[DRY-RUN] Would create namespace: %s", name)
		return nil
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err := c.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info().Msgf("Namespace already exists: %s", name)
			return nil
		}
		return fmt.Errorf("creating namespace %s: %w", name, err)
	}

	log.Info().Msgf("Created namespace: %s", name)
	return nil
}
