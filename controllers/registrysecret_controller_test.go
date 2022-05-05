package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"

	registryappsv1 "github.com/jaxwood/registry-operator/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

var _ = Describe("RegistrySecret controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		RegistrySecretName      = "test-registrysecret"
		RegistrySecretNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating RegistrySecret Status", func() {
		It("Should increase CronJob Status.Active count when new Jobs are created", func() {
			By("By creating a new RegistrySecret")
			ctx := context.Background()
			registry := &registryappsv1.RegistrySecret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps.jaxwood.com/v1",
					Kind:       "RegistrySecret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      RegistrySecretName,
					Namespace: RegistrySecretNamespace,
				},
				Spec: registryappsv1.RegistrySecretSpec{
					ImagePullSecretName: "regcred-dev",
					ImagePullSecretKey:  ".dockerconfigjson",
				},
			}
			Expect(k8sClient.Create(ctx, registry)).Should(Succeed())
			registrySecretLookupKey := types.NamespacedName{Name: RegistrySecretName, Namespace: RegistrySecretNamespace}
			createdRegistrySecret := &registryappsv1.RegistrySecret{}

			// We'll need to retry getting this newly created CronJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, registrySecretLookupKey, createdRegistrySecret)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(createdRegistrySecret.Spec.ImagePullSecretName).Should(Equal("regcred-dev"))
			Expect(createdRegistrySecret.Spec.ImagePullSecretKey).Should(Equal(".dockerconfigjson"))
		})
	})

})
