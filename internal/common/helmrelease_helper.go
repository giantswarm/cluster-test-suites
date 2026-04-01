package common

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"

	"github.com/giantswarm/clustertest/v4/pkg/logger"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testOCIRegistryURL = "oci://gsoci.azurecr.io/charts/giantswarm"
)

var helmReleaseGVK = schema.GroupVersionKind{
	Group:   "helm.toolkit.fluxcd.io",
	Version: "v2",
	Kind:    "HelmRelease",
}

func newUnstructuredHelmRelease() *unstructured.Unstructured {
	hr := &unstructured.Unstructured{}
	hr.SetGroupVersionKind(helmReleaseGVK)
	return hr
}

func newUnstructuredHelmReleaseList() *unstructured.UnstructuredList {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(helmReleaseGVK.GroupVersion().WithKind("HelmReleaseList"))
	return list
}

func ensureTestOCIRepository(ctx context.Context, c cr.Client, name, namespace, chartName string) error {
	repo := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "source.toolkit.fluxcd.io/v1beta2",
			"kind":       "OCIRepository",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"url":      fmt.Sprintf("%s/%s", testOCIRegistryURL, chartName),
				"interval": "1m",
				"ref": map[string]interface{}{
					"semver": "*",
				},
			},
		},
	}

	err := c.Create(ctx, repo)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating test OCIRepository: %w", err)
	}
	return nil
}

func deleteTestOCIRepository(ctx context.Context, c cr.Client, name, namespace string) error {
	repo := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "source.toolkit.fluxcd.io/v1beta2",
			"kind":       "OCIRepository",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	err := c.Delete(ctx, repo)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func newTestHelmRelease(name, namespace, releaseName, targetNamespace, clusterName, ociRepoName string, values map[string]interface{}) *unstructured.Unstructured {
	hr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "helm.toolkit.fluxcd.io/v2",
			"kind":       "HelmRelease",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"interval":         "1m",
				"releaseName":      releaseName,
				"targetNamespace":  targetNamespace,
				"storageNamespace": targetNamespace,
				"chartRef": map[string]interface{}{
					"kind": "OCIRepository",
					"name": ociRepoName,
				},
				"kubeConfig": map[string]interface{}{
					"secretRef": map[string]interface{}{
						"name": fmt.Sprintf("%s-kubeconfig", clusterName),
						"key":  "value",
					},
				},
				"install": map[string]interface{}{
					"createNamespace": true,
					"remediation": map[string]interface{}{
						"retries": int64(5),
					},
				},
				"values": values,
			},
		},
	}
	return hr
}

// HelmReleaseTemplateValues holds the template variables for HelmRelease values files.
type HelmReleaseTemplateValues struct {
	ClusterName string
	ExtraValues map[string]string
}

// parseValuesFile reads a YAML template file, executes Go template substitution,
// and returns the result as a map suitable for HelmRelease values.
func parseValuesFile(path string, templateValues *HelmReleaseTemplateValues) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading values file %s: %w", path, err)
	}

	tmpl, err := template.New("values").Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing values template %s: %w", path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateValues); err != nil {
		return nil, fmt.Errorf("executing values template %s: %w", path, err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &values); err != nil {
		return nil, fmt.Errorf("unmarshalling values from %s: %w", path, err)
	}

	return values, nil
}

// isHelmReleaseReady returns a function that checks if a HelmRelease has a Ready=True condition.
func isHelmReleaseReady(ctx context.Context, c cr.Client, name types.NamespacedName) func() (bool, error) {
	return func() (bool, error) {
		hr := newUnstructuredHelmRelease()
		err := c.Get(ctx, name, hr)
		if err != nil {
			return false, err
		}

		ready, reason, message := getHelmReleaseReadyCondition(hr)
		switch {
		case ready:
			logger.Log("HelmRelease '%s' is Ready", name.Name)
			return true, nil
		case reason != "":
			logger.Log("HelmRelease '%s' not yet ready: %s - %s", name.Name, reason, message)
		default:
			logger.Log("HelmRelease '%s' has no Ready condition yet", name.Name)
		}
		return false, nil
	}
}

// getHelmReleaseReadyCondition extracts the Ready condition from an unstructured HelmRelease.
func getHelmReleaseReadyCondition(hr *unstructured.Unstructured) (ready bool, reason, message string) {
	conditions, found, err := unstructured.NestedSlice(hr.Object, "status", "conditions")
	if err != nil || !found {
		return false, "", ""
	}

	for _, c := range conditions {
		condition, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _ := condition["type"].(string)
		if condType != "Ready" {
			continue
		}
		status, _ := condition["status"].(string)
		reason, _ = condition["reason"].(string)
		message, _ = condition["message"].(string)
		return status == "True", reason, message
	}
	return false, "", ""
}
