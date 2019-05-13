package coordinator

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
)

func newUnstructuredObj(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
			"spec": name,
		},
	}
}

func testClientSvr(h func(http.ResponseWriter, *http.Request)) (dynamic.Interface, *httptest.Server, error) {
	srv := httptest.NewServer(http.HandlerFunc(h))
	cl, err := dynamic.NewForConfig(&restclient.Config{
		Host: srv.URL,
	})
	if err != nil {
		srv.Close()
		return nil, nil, err
	}
	return cl, srv, nil
}

func TestK8sClient_New(t *testing.T) {
	cl, err := newK8sClient("some-ns", &restclient.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if cl == nil {
		t.Fatal("dynamic client not created	")
	}
	if cl.clientset == nil {
		t.Fatal("missing clientset")
	}
	if cl.config == nil {
		t.Fatal("missing Config")
	}
}
