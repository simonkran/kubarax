package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rs/zerolog/log"
)

// WaitForPod waits for at least one pod matching the label selector to be ready
func (c *Client) WaitForPod(ctx context.Context, namespace, labelSelector string) error {
	log.Info().Msgf("Waiting for pod with selector %s in %s", labelSelector, namespace)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for pod %s in %s: %w", labelSelector, namespace, ctx.Err())
		case <-ticker.C:
			pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Error listing pods, retrying...")
				continue
			}

			for _, pod := range pods.Items {
				for _, cond := range pod.Status.Conditions {
					if cond.Type == "Ready" && cond.Status == "True" {
						log.Info().Msgf("Pod %s is ready", pod.Name)
						return nil
					}
				}
			}
		}
	}
}

// WaitForDeployment waits for a deployment to have all replicas ready
func (c *Client) WaitForDeployment(ctx context.Context, namespace, name string) error {
	log.Info().Msgf("Waiting for deployment %s/%s to be ready", namespace, name)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for deployment %s/%s: %w", namespace, name, ctx.Err())
		case <-ticker.C:
			deploy, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				log.Debug().Err(err).Msg("Error getting deployment, retrying...")
				continue
			}

			replicas := int32(1)
			if deploy.Spec.Replicas != nil {
				replicas = *deploy.Spec.Replicas
			}
			if deploy.Status.ReadyReplicas >= replicas {
				log.Info().Msgf("Deployment %s/%s is ready (%d/%d replicas)",
					namespace, name, deploy.Status.ReadyReplicas, replicas)
				return nil
			}
		}
	}
}

// WaitForCRD waits for a CRD to be established
func (c *Client) WaitForCRD(ctx context.Context, name string) error {
	log.Info().Msgf("Waiting for CRD %s to be established", name)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for CRD %s: %w", name, ctx.Err())
		case <-ticker.C:
			// Check if the CRD exists in the API discovery
			_, err := c.clientset.Discovery().ServerResourcesForGroupVersion(name)
			if err == nil {
				log.Info().Msgf("CRD %s is established", name)
				return nil
			}
			log.Debug().Msgf("CRD %s not yet established, retrying...", name)
		}
	}
}
