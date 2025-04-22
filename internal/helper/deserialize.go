package helper

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func Deserialize(data []byte) (runtime.Object, error) {
	apiextensionsv1.AddToScheme(scheme.Scheme)      // nolint:errcheck,gosec
	apiextensionsv1beta1.AddToScheme(scheme.Scheme) // nolint:errcheck,gosec
	decoder := scheme.Codecs.UniversalDeserializer()

	runtimeObject, _, err := decoder.Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	return runtimeObject, nil
}
