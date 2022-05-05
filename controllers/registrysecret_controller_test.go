package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"

	registryappsv1 "github.com/jaxwood/registry-operator/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

var _ = Describe("RegistrySecret controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		RegistrySecretName      = "regcred-dev"
		RegistrySecretNamespace = "default"
		TestNamespace           = "foo"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating RegistrySecret Status", func() {
		It("Should sync image pull secret across namespaces", func() {
			By("By create a new image pull secret")
			ctx := context.Background()
			testSecret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      RegistrySecretName,
					Namespace: RegistrySecretNamespace,
				},
				Type: v1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte("{\"auths\":{\"registry\":{\"username\":\"foo\",\"password\":\"bar\",\"auth\":\"base64\"}}}"),
				},
			}
			Expect(k8sClient.Create(ctx, testSecret)).Should(Succeed())
			secretLookupKey := types.NamespacedName{Name: RegistrySecretName, Namespace: RegistrySecretNamespace}
			createdSecret := &v1.Secret{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretLookupKey, createdSecret)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(createdSecret.Name).Should(Equal(RegistrySecretName))
			Expect(len(createdSecret.Data[v1.DockerConfigJsonKey])).Should(Not(nil))

			By("By creating a new RegistrySecret CRDs")
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
					ImagePullSecretName: RegistrySecretName,
					ImagePullSecretKey:  v1.DockerConfigJsonKey,
				},
			}
			Expect(k8sClient.Create(ctx, registry)).Should(Succeed())
			registrySecretLookupKey := types.NamespacedName{Name: RegistrySecretName, Namespace: RegistrySecretNamespace}
			createdRegistrySecret := &registryappsv1.RegistrySecret{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, registrySecretLookupKey, createdRegistrySecret)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(createdRegistrySecret.Spec.ImagePullSecretName).Should(Equal(RegistrySecretName))
			Expect(createdRegistrySecret.Spec.ImagePullSecretKey).Should(Equal(v1.DockerConfigJsonKey))

			By("By creating a new namespace")
			testNamespace := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: TestNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, testNamespace)).Should(Succeed())
			namespaceKey := types.NamespacedName{Name: TestNamespace, Namespace: TestNamespace}
			namespace := &v1.Namespace{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespaceKey, namespace)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(namespace.Name).Should(Equal(TestNamespace))

			By("By syncing the image pull secret")
			syncedSecretLookupKey := types.NamespacedName{Name: RegistrySecretName, Namespace: TestNamespace}
			syncedCreatedSecret := &v1.Secret{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, syncedSecretLookupKey, syncedCreatedSecret)
				fmt.Fprintf(GinkgoWriter, "[DEBUG] OUTPUT LINE: %s\n", err)
				if err != nil {
					return false
				}
				return true
			}, timeout, duration).Should(BeTrue())

			Expect(syncedCreatedSecret.Name).Should(Equal(RegistrySecretName))
			Expect(len(syncedCreatedSecret.Data[v1.DockerConfigJsonKey])).Should(Not(nil))
		})
	})

})
