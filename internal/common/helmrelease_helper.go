package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
	"time"

	helm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/giantswarm/clustertest/v4/pkg/logger"
	"gopkg.in/yaml.v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testHelmRepoName = "giantswarm-catalog-test"
	testHelmRepoURL  = "https://giantswarm.github.io/giantswarm-catalog/"
)

func ensureTestHelmRepository(ctx context.Context, c cr.Client, namespace string) error {
	repo := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "source.toolkit.fluxcd.io/v1",
			"kind":       "HelmRepository",
			"metadata": map[string]interface{}{
				"name":      testHelmRepoName,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"url":      testHelmRepoURL,
				"interval": "10m",
			},
		},
	}

	err := c.Create(ctx, repo)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating test HelmRepository: %w", err)
	}
	return nil
}

func deleteTestHelmRepository(ctx context.Context, c cr.Client, namespace string) error {
	repo := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "source.toolkit.fluxcd.io/v1",
			"kind":       "HelmRepository",
			"metadata": map[string]interface{}{
				"name":      testHelmRepoName,
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

func newTestHelmRelease(name, namespace, chartName, releaseName, targetNamespace, clusterName string, values map[string]interface{}) (*helm.HelmRelease, error) {
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshalling helm values: %w", err)
	}

	return &helm.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: helm.HelmReleaseSpec{
			Interval:        metav1.Duration{Duration: 1 * time.Minute},
			ReleaseName:     releaseName,
			TargetNamespace: targetNamespace,
			Chart: &helm.HelmChartTemplate{
				Spec: helm.HelmChartTemplateSpec{
					Chart:   chartName,
					Version: "*",
					SourceRef: helm.CrossNamespaceObjectReference{
						Kind: "HelmRepository",
						Name: testHelmRepoName,
					},
				},
			},
			KubeConfig: &meta.KubeConfigReference{
				SecretRef: &meta.SecretKeyReference{
					Name: fmt.Sprintf("%s-kubeconfig", clusterName),
					Key:  "value",
				},
			},
			Install: &helm.Install{
				CreateNamespace: true,
				Remediation: &helm.InstallRemediation{
					Retries: 5,
				},
			},
			Values: &apiextensionsv1.JSON{Raw: valuesJSON},
		},
	}, nil
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

func isHelmReleaseReady(ctx context.Context, c cr.Client, name types.NamespacedName) func() (bool, error) {
	return func() (bool, error) {
		hr := &helm.HelmRelease{}
		err := c.Get(ctx, name, hr)
		if err != nil {
			return false, err
		}

		for _, condition := range hr.Status.Conditions {
			if condition.Type == "Ready" {
				if condition.Status == metav1.ConditionTrue {
					logger.Log("HelmRelease '%s' is Ready", name.Name)
					return true, nil
				}
				logger.Log("HelmRelease '%s' not yet ready: %s - %s", name.Name, condition.Reason, condition.Message)
				return false, nil
			}
		}

		logger.Log("HelmRelease '%s' has no Ready condition yet", name.Name)
		return false, nil
	}
}
