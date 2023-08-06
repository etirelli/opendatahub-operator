/*
Copyright 2023.

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

package dscinitialization

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dsci "github.com/opendatahub-io/opendatahub-operator/v2/apis/dscinitialization/v1alpha1"
	"github.com/opendatahub-io/opendatahub-operator/v2/controllers/status"
	"github.com/opendatahub-io/opendatahub-operator/v2/pkg/deploy"
	ofapi "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("DSCInitialization controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		dscApplicationsNamespace = "opendatahub"
		defaultDSCIName          = "default"
		operatorsNamespace       = "openshift-operators"
		timeout                  = time.Second * 10
		duration                 = time.Second * 10
		interval                 = time.Millisecond * 250
	)

	Context("When updating DSCInitialization Status", func() {
		It("Should be a singleton", func() {
			By("By creating a new DSCInitialization")
			ctx := context.Background()

			/*
				    The operator checks for the CSV to detect the target platform.
					We will emulate the ODH platform
			*/
			namespace := &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: operatorsNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

			csv := &ofapi.ClusterServiceVersion{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterServiceVersion",
					APIVersion: "operators.coreos.com/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "opendatahub-operator.v0.0.1",
					Namespace: operatorsNamespace,
				},
				Spec: ofapi.ClusterServiceVersionSpec{
					DisplayName: string(deploy.OpenDataHub),
					InstallStrategy: ofapi.NamedInstallStrategy{
						StrategyName: "deployment",
					},
				},
			}
			Expect(k8sClient.Create(ctx, csv)).Should(Succeed())

			defaultDSCInit := &dsci.DSCInitialization{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DSCInitialization",
					APIVersion: "dscinitialization.opendatahub.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: defaultDSCIName,
				},
				Spec: dsci.DSCInitializationSpec{
					ApplicationsNamespace: dscApplicationsNamespace,
					Monitoring: dsci.Monitoring{
						Enabled: false,
					},
				},
			}
			Expect(k8sClient.Create(ctx, defaultDSCInit)).Should(Succeed())

			/*
				After creating the default DSCInitilization, we should verify that it is the only DSCInitialization in the cluster.
			*/
			lookupKey := types.NamespacedName{Name: defaultDSCIName, Namespace: dscApplicationsNamespace}
			createdDSCI := &dsci.DSCInitialization{}

			// We'll need to retry getting this newly created CronJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdDSCI)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("By checking the DSCInitialization is ready")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, lookupKey, createdDSCI)
				if err != nil {
					return "", err
				}
				return createdDSCI.Status.Phase, nil
			}, duration, interval).Should(Equal(status.PhaseReady))

			/*
				Adding this Job to our test CronJob should trigger our controller’s reconciler logic.
				After that, we can write a test that evaluates whether our controller eventually updates our CronJob’s Status field as expected!
			*/
			// By("By checking that the CronJob has one active Job")
			// Eventually(func() ([]string, error) {
			// 	err := k8sClient.Get(ctx, cronjobLookupKey, createdCronjob)
			// 	if err != nil {
			// 		return nil, err
			// 	}

			// 	names := []string{}
			// 	for _, job := range createdCronjob.Status.Active {
			// 		names = append(names, job.Name)
			// 	}
			// 	return names, nil
			// }, timeout, interval).Should(ConsistOf(JobName), "should list our active job %s in the active jobs list in status", JobName)
		})
	})

})
