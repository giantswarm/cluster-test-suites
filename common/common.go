package common

import (
	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Run(framework *clustertest.Framework, cluster *application.Cluster) {
	var wcClient *client.Client

	BeforeEach(func() {
		var err error

		wcClient, err = framework.WC(cluster.Name)
		if err != nil {
			Fail(err.Error())
		}
	})

	It("should be able to connect to MC cluster", func() {
		Expect(framework.MC().CheckConnection()).To(Succeed())
	})

	It("should be able to connect to WC cluster", func() {
		Expect(wcClient.CheckConnection()).To(Succeed())
	})
}
