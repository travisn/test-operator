package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

const (
	tprName    = "test"
	tprGroup   = "myrook.io"
	tprVersion = "v1beta1"
)

func main() {
	ns := os.Getenv("MY_POD_NAMESPACE")
	if ns == "" {
		log.Fatalf("failed to specify namespace")
	}
	log.Printf("Testing watching TPRs in namespace %s", ns)

	host, clientset, err := getClientset()
	if err != nil {
		log.Fatalf("failed to get k8s client. %+v", err)
	}

	err = createTPR(clientset, ns)
	if err != nil {
		log.Fatalf("failed to create TPR. %+v", err)
	}

	watchTPR(clientset, ns, host)

	log.Printf("waiting for log analysis...")
	<-time.After(24 * time.Hour)
}

func watchTPR(clientset *kubernetes.Clientset, ns, host string) {
	log.Printf("start watching test tpr")
	resourceVersion := "0"

	httpCli, err := newHttpClient()
	if err != nil {
		log.Fatalf("failed to get http tpr client. %+v", err)
	}

	uri := fmt.Sprintf("%s/apis/%s/%s/namespaces/%s/%s?watch=true&resourceVersion=%s",
		host, tprGroup, tprVersion, ns, tprName, resourceVersion)
	log.Printf("watching uri: %s", uri)
	resp, err := httpCli.Client.Get(uri)
	if err != nil {
		log.Fatalf("failed to watch TPR. %+v", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("failed to watch TPR. %+v", resp)
	}

	log.Printf("HOORAY! received TPR event: %+v", resp)
}

func createTPR(clientset *kubernetes.Clientset, ns string) error {
	log.Printf("creating test TPR")
	tpr := &v1beta1.ThirdPartyResource{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", tprName, tprGroup),
		},
		Versions: []v1beta1.APIVersion{
			{Name: tprVersion},
		},
		Description: "test TPR",
	}
	_, err := clientset.ExtensionsV1beta1().ThirdPartyResources().Create(tpr)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create rook third party resources. %+v", err)
		}
	}

	// brain-dead wait
	for i := 0; i < 10; i++ {
		log.Printf("%d: waiting for tpr init", i)
		<-time.After(time.Second)

		uri := fmt.Sprintf("/apis/%s/%s/namespaces/%s/clusters", tprGroup, tprVersion, ns)
		_, err := clientset.CoreV1().RESTClient().Get().RequestURI(uri).DoRaw()
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("tpr not found")
				continue
			}
			return err
		}
	}
	return nil
}
