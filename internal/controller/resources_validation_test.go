/*
Copyright 2026 Valkey Contributors.

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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	valkeyiov1alpha1 "github.com/valkey-io/valkey-operator/api/v1alpha1"
)

func resourcesCluster(name string, requests, limits corev1.ResourceList) *valkeyiov1alpha1.ValkeyCluster {
	return &valkeyiov1alpha1.ValkeyCluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: valkeyiov1alpha1.ValkeyClusterSpec{
			Shards:    1,
			Resources: corev1.ResourceRequirements{Requests: requests, Limits: limits},
		},
	}
}

// Requests above limits are rejected by the API server only when the operator
// later writes the StatefulSet, leaving the cluster stuck. Reject them up front.
var _ = Describe("resources CEL validation", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	q := resource.MustParse

	accepted := map[string]*valkeyiov1alpha1.ValkeyCluster{
		"no resources":      resourcesCluster("res-ok-a", nil, nil),
		"requests only":     resourcesCluster("res-ok-b", corev1.ResourceList{corev1.ResourceMemory: q("1Gi")}, nil),
		"limits only":       resourcesCluster("res-ok-c", nil, corev1.ResourceList{corev1.ResourceCPU: q("1")}),
		"requests = limits": resourcesCluster("res-ok-d", corev1.ResourceList{corev1.ResourceMemory: q("1Gi"), corev1.ResourceCPU: q("1")}, corev1.ResourceList{corev1.ResourceMemory: q("1024Mi"), corev1.ResourceCPU: q("1000m")}),
		"requests < limits": resourcesCluster("res-ok-e", corev1.ResourceList{corev1.ResourceMemory: q("512Mi"), corev1.ResourceCPU: q("500m")}, corev1.ResourceList{corev1.ResourceMemory: q("1Gi"), corev1.ResourceCPU: q("1")}),
	}
	rejected := map[string]struct {
		cluster *valkeyiov1alpha1.ValkeyCluster
		message string
	}{
		"memory requests > limits": {
			resourcesCluster("res-bad-mem", corev1.ResourceList{corev1.ResourceMemory: q("2Gi")}, corev1.ResourceList{corev1.ResourceMemory: q("1Gi")}),
			"resources.requests.memory must be less than or equal to resources.limits.memory",
		},
		"cpu requests > limits": {
			resourcesCluster("res-bad-cpu", corev1.ResourceList{corev1.ResourceCPU: q("2")}, corev1.ResourceList{corev1.ResourceCPU: q("1500m")}),
			"resources.requests.cpu must be less than or equal to resources.limits.cpu",
		},
	}

	for desc, c := range accepted {
		It("accepts "+desc, func() {
			Expect(k8sClient.Create(ctx, c)).To(Succeed())
			Expect(k8sClient.Delete(ctx, c)).To(Succeed())
		})
	}

	for desc, tc := range rejected {
		It("rejects "+desc, func() {
			err := k8sClient.Create(ctx, tc.cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(tc.message))
		})
	}

	It("rejects an update that lowers memory limits below requests", func() {
		c := resourcesCluster("res-update", corev1.ResourceList{corev1.ResourceMemory: q("1Gi")}, corev1.ResourceList{corev1.ResourceMemory: q("2Gi")})
		Expect(k8sClient.Create(ctx, c)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, c)).To(Succeed()) })

		c.Spec.Resources.Limits[corev1.ResourceMemory] = q("512Mi")
		err := k8sClient.Update(ctx, c)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("resources.requests.memory must be less than or equal to resources.limits.memory"))
	})
})
