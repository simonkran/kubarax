package k8s

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"

	"github.com/rs/zerolog/log"
)

// ApplyOptions configures server-side apply behavior
type ApplyOptions struct {
	FieldManager   string
	ForceConflicts bool
	DryRun         bool
}

// DefaultApplyOptions returns sensible defaults for apply operations
func DefaultApplyOptions() ApplyOptions {
	return ApplyOptions{
		FieldManager:   "kubarax",
		ForceConflicts: true,
		DryRun:         false,
	}
}

// ApplyManifest applies a multi-document YAML manifest using server-side apply
func (c *Client) ApplyManifest(ctx context.Context, manifest []byte, opts ApplyOptions) error {
	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(manifest)))

	for {
		doc, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("reading YAML document: %w", err)
		}

		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}

		obj := &unstructured.Unstructured{}
		if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(doc), len(doc)).Decode(obj); err != nil {
			return fmt.Errorf("decoding YAML document: %w", err)
		}

		if obj.GetKind() == "" {
			continue
		}

		if err := c.applyObject(ctx, obj, opts); err != nil {
			return fmt.Errorf("applying %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}
	}

	return nil
}

// applyObject applies a single unstructured object
func (c *Client) applyObject(ctx context.Context, obj *unstructured.Unstructured, opts ApplyOptions) error {
	gvk := obj.GroupVersionKind()

	mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("finding REST mapping for %v: %w", gvk, err)
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ns := obj.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		dr = c.dynamicClient.Resource(mapping.Resource).Namespace(ns)
	} else {
		dr = c.dynamicClient.Resource(mapping.Resource)
	}

	applyOpts := metav1.ApplyOptions{
		FieldManager: opts.FieldManager,
		Force:        opts.ForceConflicts,
	}

	if opts.DryRun {
		applyOpts.DryRun = []string{metav1.DryRunAll}
	}

	obj.SetManagedFields(nil)

	_, err = dr.Apply(ctx, obj.GetName(), obj, applyOpts)
	if err != nil {
		return fmt.Errorf("server-side apply: %w", err)
	}

	log.Debug().Msgf("Applied %s/%s in %s", obj.GetKind(), obj.GetName(), obj.GetNamespace())
	return nil
}
