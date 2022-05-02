/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	cachev1appsv1 "github.com/jaxwood/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// RegistrySecretReconciler reconciles a RegistrySecret object
type RegistrySecretReconciler struct {
	Client   client.Client
	Lister   client.Reader
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=apps.jaxwood.com,resources=registrysecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.jaxwood.com,resources=registrysecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.jaxwood.com,resources=registrysecrets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=secrets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RegistrySecret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *RegistrySecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling RegistrySecret")

	var registry cachev1appsv1.RegistrySecret
	err := r.Lister.Get(ctx, req.NamespacedName, &registry)

	if apierrors.IsNotFound(err) {
		log.Info("RegistrySecret no longer exists, ignoring")
		return ctrl.Result{}, nil
	}

	if err != nil {
		log.Error(err, "failed to get RegistrySecret")
		return ctrl.Result{}, fmt.Errorf("failed to get %q: %s", req.NamespacedName, err)
	}

	var namespaceList corev1.NamespaceList
	if err := r.Lister.List(ctx, &namespaceList); err != nil {
		log.Error(err, "failed to list namespaces")
		r.Recorder.Eventf(&registry, corev1.EventTypeWarning, "NamespaceListError", "Failed to list namespaces: %s", err)
		return ctrl.Result{}, fmt.Errorf("failed to list Namespaces: %w", err)
	}

	data, _ := r.getRegistrySecret(ctx, &registry)

	for _, namespace := range namespaceList.Items {
		log = log.WithValues("namespace", namespace.Name)

		// Don't reconcile target for Namespaces that are being terminated.
		if namespace.Status.Phase == corev1.NamespaceTerminating {
			log.WithValues("phase", corev1.NamespaceTerminating).Info("skipping sync for namespace as it is terminating")
			continue
		}

		_, err := r.syncSecret(ctx, log, &registry, namespace.Name, data)
		if err != nil {
			log.Error(err, "failed sync RegistrySecret to target namespace")
			r.Recorder.Eventf(&registry, corev1.EventTypeWarning, "SyncTargetFailed", "Failed to sync target in Namespace %q: %s", namespace.Name, err)

			return ctrl.Result{Requeue: true}, r.Client.Status().Update(ctx, &registry)
		}
	}

	log.Info("successfully synced RegistrySecret")

	r.Recorder.Eventf(&registry, corev1.EventTypeNormal, "Synced", "Successfully synced RegistrySecret to all namespaces")
	return ctrl.Result{}, r.Client.Status().Update(ctx, &registry)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegistrySecretReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	log := log.FromContext(ctx)
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1appsv1.RegistrySecret{}).
		Watches(&source.Kind{Type: new(corev1.Namespace)}, handler.EnqueueRequestsFromMapFunc(
			func(obj client.Object) []reconcile.Request {
				err := r.Lister.Get(ctx, client.ObjectKeyFromObject(obj), obj)
				if apierrors.IsNotFound(err) {
					// No need to reconcile all RegistrySecret if the namespace has been deleted.
					log.Info(fmt.Sprintf("failed to find any RegistrySecret in the namespace %s", (obj.(*corev1.Namespace)).Name))
					return nil
				}

				if err != nil {
					// If an error occurred we still need to reconcile all RegistrySecret to
					// ensure no Namespace gets lost.
					log.Error(err, "failed to get Namespace, reconciling all RegistrySecret anyway", "namespace", obj.GetName())
				}

				// If an error happens here and we do nothing, we run the risk of
				// leaving a Namespace behind when syncing.
				// Exiting error is the safest option, as it will force a resync on
				// all RegistrySecret on start.
				var registrySecretList cachev1appsv1.RegistrySecretList
				if err := r.Lister.List(ctx, &registrySecretList); err != nil {
					log.Error(err, "failed to list all RegistrySecrets, exiting error")
					os.Exit(-1)
				}

				var requests []reconcile.Request
				for _, registrySecret := range registrySecretList.Items {
					requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
						Name: registrySecret.Name, Namespace: registrySecret.Namespace,
					}})
				}

				return requests
			},
		)).
		Complete(r)
}

type notFoundError struct{ error }

// getRegistrySecret returns the data in the target Secret within the namespace.
func (r *RegistrySecretReconciler) getRegistrySecret(ctx context.Context, ref *cachev1appsv1.RegistrySecret) (string, error) {
	var secret corev1.Secret
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: ref.Namespace, Name: ref.Spec.ImagePullSecretName}, &secret)
	if apierrors.IsNotFound(err) {
		return "", notFoundError{err}
	}
	if err != nil {
		return "", fmt.Errorf("failed to get Secret %s/%s: %w", ref.Spec.GetNamespace(), ref.Name, err)
	}

	data, ok := secret.Data[ref.Spec.ImagePullSecretKey]
	if !ok {
		return "", notFoundError{fmt.Errorf("no data found in Secret %s/%s at key %q", ref.Spec.GetNamespace(), ref.Name, ref.DeepCopy().Spec.ImagePullSecretKey)}
	}

	return string(data), nil
}

// syncSecret syncs the given data to the target Secret in the given
// namespace.
// The name of the Secret is the same as the RegistrySecret.
// Ensures the Secret is owned by the given Registrysecret, and the data is up to
// date.
// Returns true if the Secret has been created or was updated.
func (b *RegistrySecretReconciler) syncSecret(ctx context.Context, log logr.Logger, registrySecret *cachev1appsv1.RegistrySecret, namespace, data string) (bool, error) {
	var secret corev1.Secret
	err := b.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: registrySecret.Name}, &secret)

	// If the Secret doesn't exist yet, create it
	if apierrors.IsNotFound(err) {
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:            registrySecret.Name,
				Namespace:       namespace,
				OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(registrySecret, cachev1appsv1.SchemeGroupVersion.WithKind("SecretRegistry"))},
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(data),
			},
		}

		return true, b.Client.Create(ctx, &secret)
	}

	if err != nil {
		return false, fmt.Errorf("failed to get secret %s/%s: %w", namespace, registrySecret.Name, err)
	}

	var needsUpdate bool

	// If Secret is missing OwnerReference, add it back.
	if !metav1.IsControlledBy(&secret, registrySecret) {
		secret.OwnerReferences = append(secret.OwnerReferences, *metav1.NewControllerRef(registrySecret, cachev1appsv1.SchemeGroupVersion.WithKind("RegistrySecret")))
		needsUpdate = true
	}

	// Match, return do nothing
	secretValue, ok := secret.Data[registrySecret.Spec.ImagePullSecretKey]
	if !ok || string(secretValue) != data {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[registrySecret.Spec.ImagePullSecretKey] = []byte(data)
		needsUpdate = true
	}

	// Exit early if no update is needed
	if !needsUpdate {
		return false, nil
	}

	if err := b.Client.Update(ctx, &secret); err != nil {
		return true, fmt.Errorf("failed to update secret %s/%s with RegistrySecret: %w", namespace, registrySecret.Name, err)
	}

	log.Info("synced RegistrySecret to namespace")

	return true, nil
}
