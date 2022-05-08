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
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/rh-mobb/ecr-secret-operator/api/v1alpha1"
	"github.com/rh-mobb/ecr-secret-operator/ecr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const iamSecretName string = "aws-ecr-cloud-credentials"

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	SecretGenerator ecr.SecretGenerator
}

//+kubebuilder:rbac:groups=ecr.mobb.redhat.com,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ecr.mobb.redhat.com,resources=secrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ecr.mobb.redhat.com,resources=secrets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Secret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("Reconciling ECR Secret")

	//Get CRD secret object
	ecrSecret := &v1alpha1.Secret{}
	err := r.Client.Get(ctx, req.NamespacedName, ecrSecret)
	if err != nil {
		reqLogger.Error(err, fmt.Sprintf("Can not find ECR Secret %s/%s", req.Namespace, req.NamespacedName))
		if errors.IsNotFound(err) {
			return reconcile.Result{}, client.IgnoreNotFound(err)
		}
		return reconcile.Result{}, err
	}

	reqLogger.Info("Generate ECR token")
	newSecret, err := r.SecretGenerator.GenerateSecret(&ecr.Input{
		S: ecrSecret,
	})
	if err != nil {
		reqLogger.Error(err, "can not generate secret")
		return reconcile.Result{}, err
	}
	secret := &v1.Secret{}
	var message string
	// Generate a new secret
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: ecrSecret.Spec.GenerateSecretName}, secret); err != nil {
		if errors.IsNotFound(err) {
			message = fmt.Sprintf("Generate new secret %s/%s", req.Namespace, ecrSecret.Spec.GenerateSecretName)
			reqLogger.Info(message)
			if err = r.Client.Create(ctx, newSecret); err != nil {
				return reconcile.Result{}, err
			}
			ecrSecret.Status.Phase = "Created"
		} else {
			return ctrl.Result{}, err
		}
	} else {
		message = fmt.Sprintf("Update secret %s/%s", req.Namespace, ecrSecret.Spec.GenerateSecretName)
		reqLogger.Info(message)
	}
	secret.Data = newSecret.Data
	if err = r.Client.Update(ctx, secret); err != nil {
		return ctrl.Result{}, err
	}

	ecrSecret.Status.Phase = "Updated"
	ecrSecret.Status.LastUpdatedTime = &metav1.Time{Time: time.Now()}
	// ecrSecret.Status.Conditions.
	if err := r.Client.Status().Update(ctx, ecrSecret); err != nil {
		reqLogger.Error(err, "unable to update ECR secret status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: ecrSecret.Spec.Frequency.Duration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Secret{}).
		Complete(r)
}
