package controllers

import (
	"context"

	pkgerr "github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
)

func (r *MilvusClusterReconciler) updateService(
	mc v1alpha1.MilvusCluster, service *corev1.Service, component MilvusComponent,
) error {
	appLabels := NewComponentAppLabels(mc.Name, component.String())
	service.Labels = MergeLabels(service.Labels, appLabels)
	if err := ctrl.SetControllerReference(&mc, service, r.Scheme); err != nil {
		return err
	}

	// we only have proxy service now
	// if component.Name == Proxy.Name {
	service.Labels = MergeLabels(service.Labels, mc.Spec.Com.Proxy.ServiceLabels)
	service.Annotations = MergeLabels(service.Annotations, mc.Spec.Com.Proxy.ServiceAnnotations)
	// }

	service.Spec.Ports = MergeServicePort(service.Spec.Ports, component.GetServicePorts(mc.Spec))
	service.Spec.Selector = appLabels
	service.Spec.Type = component.GetServiceType(mc.Spec)

	return nil
}

func (r *MilvusClusterReconciler) ReconcileComponentService(
	ctx context.Context, mc v1alpha1.MilvusCluster, component MilvusComponent,
) error {
	if component.IsNode() || component.IsCoord() {
		return nil
	}

	namespacedName := NamespacedName(mc.Namespace, GetServiceInstanceName(mc.Name))
	old := &corev1.Service{}
	err := r.Get(ctx, namespacedName, old)
	if errors.IsNotFound(err) {
		new := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}
		if err := r.updateService(mc, new, component); err != nil {
			return err
		}

		r.logger.Info("Create Service", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateService(mc, cur, component); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		return nil
	}

	/* if config.IsDebug() {
		diff, err := diffObject(old, cur)
		if err == nil {
			r.logger.Info("Service diff", "name", cur.Name, "namespace", cur.Namespace, "diff", string(diff))
		}
	} */

	r.logger.Info("Update Service", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}

func (r *MilvusClusterReconciler) ReconcileServices(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	err := r.ReconcileComponentService(ctx, mc, Proxy)
	return pkgerr.Wrap(err, "reconcile milvus services")
}

func (r *MilvusReconciler) ReconcileServices(ctx context.Context, mil v1alpha1.Milvus) error {
	old := &corev1.Service{}
	key := client.ObjectKey{
		Namespace: mil.Namespace,
		Name:      GetServiceInstanceName(mil.Name),
	}
	err := r.Get(ctx, key, old)
	if errors.IsNotFound(err) {
		new := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetServiceInstanceName(mil.Name),
				Namespace: mil.Namespace,
			},
		}
		if err := r.updateService(mil, new); err != nil {
			return err
		}

		r.logger.Info("Create Service", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateService(mil, cur); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		return nil
	}

	r.logger.Info("Update Service", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}

func (r *MilvusReconciler) updateService(
	mc v1alpha1.Milvus, service *corev1.Service,
) error {
	appLabels := NewComponentAppLabels(mc.Name, MilvusName)
	service.Labels = MergeLabels(service.Labels, appLabels)
	if err := ctrl.SetControllerReference(&mc, service, r.Scheme); err != nil {
		return err
	}

	service.Labels = MergeLabels(service.Labels, mc.Spec.ServiceLabels)
	service.Annotations = MergeLabels(service.Annotations, mc.Spec.ServiceAnnotations)

	service.Spec.Ports = MergeServicePort(service.Spec.Ports, []corev1.ServicePort{
		{
			Name:       MilvusName,
			Protocol:   corev1.ProtocolTCP,
			Port:       MilvusPort,
			TargetPort: intstr.FromString(MilvusName),
		},
		{
			Name:       MetricPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       MetricPort,
			TargetPort: intstr.FromString(MetricPortName),
		},
	})

	service.Spec.Selector = appLabels
	service.Spec.Type = mc.Spec.ServiceType

	return nil
}
