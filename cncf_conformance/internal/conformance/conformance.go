package conformance

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/giantswarm/cluster-test-suites/cncf_conformance/internal/sonobuoy"
	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"
)

var (
	Framework *clustertest.Framework
	Cluster   *application.Cluster
)

func Run() {
	var (
		wcKubeconfig   string
		sonobuoyClient *sonobuoy.Client
	)

	It("has all the control-plane nodes running", func() {
		wcClient, err := Framework.WC(Cluster.Name)
		Expect(err).NotTo(HaveOccurred())

		Eventually(wait.AreNumNodesReady(context.Background(), wcClient, 1, &cr.MatchingLabels{"node-role.kubernetes.io/control-plane": ""})).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})

	It("has all the worker nodes running", func() {
		wcClient, err := Framework.WC(Cluster.Name)
		Expect(err).NotTo(HaveOccurred())

		Eventually(wait.AreNumNodesReady(context.Background(), wcClient, 1, client.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"})).
			WithTimeout(wait.DefaultTimeout).
			WithPolling(wait.DefaultInterval).
			Should(Succeed())
	})

	It("should store the WC kubeconfig", func() {
		file, err := os.CreateTemp("", "kubeconfig-")
		Expect(err).NotTo(HaveOccurred())
		defer file.Close()

		kubeconfig, err := Framework.MC().GetClusterKubeConfig(context.Background(), Cluster.Name, Cluster.Namespace)
		Expect(err).NotTo(HaveOccurred())

		_, err = file.Write([]byte(kubeconfig))
		Expect(err).NotTo(HaveOccurred())

		wcKubeconfig = file.Name()
	})

	It("should create a Sonobuoy client", func() {
		sonobuoyClient = sonobuoy.New(wcKubeconfig)
	})

	It("should have the required binaries available", func() {
		Expect(sonobuoyClient.BinaryExists()).To(BeTrue())
	})

	It("should run the CNCF conformance tests", func() {
		err := sonobuoyClient.RunTests("certified-conformance")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should save the conformance report", func() {
		err := sonobuoyClient.SaveResults()
		if err != nil {
			AbortSuite("Failed to save the conformance test results, aborting test suite...")
		}

		GinkgoWriter.Printf("Saved results to: %s", os.Getenv("RESULTS_DIRECTORY"))
	})

	It("should have passed all conformance tests", func() {
		failedTests := sonobuoyClient.GetFailed()
		if len(failedTests) > 0 {
			By("Failed conformance tests:")
			for _, test := range failedTests {
				GinkgoWriter.Printf("\t- %s", test)
			}
		}

		Expect(len(failedTests)).To(BeZero())
		Expect(sonobuoyClient.HasPassed()).To(BeTrue())
	})

	It("should store metadata about conformance run", func() {
		data, err := yaml.Marshal(map[string]string{
			"vendor":                "Giant Swarm",
			"name":                  fmt.Sprintf("Managed Kubernetes on %s", strings.TrimPrefix(Cluster.ClusterApp.AppName, "cluster-")),
			"version":               Cluster.ClusterApp.Version,
			"website_url":           "https://giantswarm.io",
			"repo_url":              fmt.Sprintf("https://github.com/giantswarm/%s/", Cluster.ClusterApp.AppName),
			"documentation_url":     "https://docs.giantswarm.io",
			"product_logo_url":      "https://raw.githubusercontent.com/giantswarm/brand/v1.0.0/logo/special-purpose/cncf-landscape-stacked.svg",
			"type":                  "distribution",
			"description":           "The Giant Swarm platform enables users to simply and rapidly create and use 24/7 managed Kubernetes clusters on-demand.",
			"contact_email_address": "info@giantswarm.io",
		})
		Expect(err).To(BeNil())

		err = os.WriteFile(fmt.Sprintf("%sPRODUCT.yaml", os.Getenv("RESULTS_DIRECTORY")), data, 0666)
		Expect(err).To(BeNil())

	})
}
