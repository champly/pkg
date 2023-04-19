package authentication

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/symcn/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type CertInfo struct {
	// client verify Certificate
	CABundle []byte

	// server load
	TLSKey  []byte
	TLSCert []byte
}

type SignedWay string

var (
	SelfSigned SignedWay = "SelfSigned"
	CSRSigned  SignedWay = "CSRSigned"
)

// SaveTLSToDir save TLSKey and TLSCert to path
// filename is tls.key and tls.crt
func (ci *CertInfo) SaveTLSToPath(path string) error {
	_, err := os.Stat(path)
	if err != nil && !os.IsExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	err = os.WriteFile(fmt.Sprintf("%s/tls.crt", path), ci.TLSCert, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(fmt.Sprintf("%s/tls.key", path), ci.TLSKey, 0644)
	if err != nil {
		return err
	}
	return nil
}

// UpdateCABundleToMutatingWebhook update CABundle to MutatingWebhookConfigurations
// use this way need those rules:
//   - apiGroups: ["admissionregistration.k8s.io"]
//     resources: ["mutatingwebhookconfigurations"]
//     verbs: ["get", "update"]
func (ci *CertInfo) UpdateCABundleToMutatingWebhook(client api.MingleClient, mutatingName, svcName, svcNamespace string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()

	// update mutatingwebhookconfiguration caBundle
	mutatingClient := client.GetKubeInterface().AdmissionregistrationV1().MutatingWebhookConfigurations()
	mutating, err := mutatingClient.Get(ctx, mutatingName, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "get MutatingWebhookConfigurations failed", "name", mutatingName)
		return err
	}

	var change = 0
	for i := range mutating.Webhooks {
		if mutating.Webhooks[i].ClientConfig.Service.Namespace == svcNamespace &&
			mutating.Webhooks[i].ClientConfig.Service.Name == svcName {

			mutating.Webhooks[i].ClientConfig.CABundle = ci.CABundle
			klog.V(4).Infof("modifiy MutatingWebhookConfigurations (%s) webhook's %s caBundle.", mutatingName, mutating.Webhooks[i].Name)
			change++
		}
	}
	if change == 0 {
		return fmt.Errorf("not found MutatingWebhookConfigurations (%s) match svc(%s/%s) info",
			mutatingName,
			svcNamespace,
			svcName,
		)
	}

	_, err = mutatingClient.Update(ctx, mutating, metav1.UpdateOptions{})
	if err != nil {
		klog.ErrorS(err, "update MutatingWebhookConfigurations failed", "name", mutatingName)
		return err
	}

	klog.InfoS("update MutatingWebhookName success.", "name", mutatingName, "update webhook count", change)
	return nil
}
