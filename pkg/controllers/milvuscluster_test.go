package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/helm"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCluster_Finalize(t *testing.T) {
	env := newClusterTestEnv(t)
	r := env.Reconciler
	mockClient := env.MockClient
	ctx := env.ctx
	m := env.Inst
	mockHelm := helm.NewMockClient(env.Ctrl)
	helm.SetDefaultClient(mockHelm)
	errTest := errors.New("test")

	t.Run("no delete", func(t *testing.T) {
		err := r.Finalize(ctx, m)
		assert.NoError(t, err)
	})

	t.Run("etcd delete pvc", func(t *testing.T) {
		defer env.checkMocks()
		m.Spec.Dep.Etcd.InCluster.DeletionPolicy = v1alpha1.DeletionPolicyDelete
		m.Spec.Dep.Etcd.InCluster.PVCDeletion = true
		mockHelm.EXPECT().Uninstall(gomock.Any(), gomock.Any())
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any())
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
				pvcList := list.(*corev1.PersistentVolumeClaimList)
				pvcList.Items = []corev1.PersistentVolumeClaim{
					{},
				}
			})
		mockClient.EXPECT().Delete(gomock.Any(), gomock.Any())
		err := r.Finalize(ctx, m)
		assert.NoError(t, err)
	})

	t.Run("pulsar delete pvc", func(t *testing.T) {
		defer env.checkMocks()
		m.Spec.Dep.Etcd.InCluster.DeletionPolicy = v1alpha1.DeletionPolicyRetain
		m.Spec.Dep.Pulsar.InCluster.DeletionPolicy = v1alpha1.DeletionPolicyDelete
		m.Spec.Dep.Pulsar.InCluster.PVCDeletion = true
		mockHelm.EXPECT().Uninstall(gomock.Any(), gomock.Any())
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any())
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
				pvcList := list.(*corev1.PersistentVolumeClaimList)
				pvcList.Items = []corev1.PersistentVolumeClaim{
					{},
				}
			})
		mockClient.EXPECT().Delete(gomock.Any(), gomock.Any())
		err := r.Finalize(ctx, m)
		assert.NoError(t, err)
	})

	t.Run("storage uninstall failed", func(t *testing.T) {
		defer env.checkMocks()
		m.Spec.Dep.Pulsar.InCluster.DeletionPolicy = v1alpha1.DeletionPolicyRetain
		m.Spec.Dep.Storage.InCluster.DeletionPolicy = v1alpha1.DeletionPolicyDelete
		m.Spec.Dep.Storage.InCluster.PVCDeletion = true
		mockHelm.EXPECT().Uninstall(gomock.Any(), gomock.Any()).Return(errTest)
		err := r.Finalize(ctx, m)
		assert.Error(t, err)
	})

	t.Run("storage list failed", func(t *testing.T) {
		defer env.checkMocks()
		mockHelm.EXPECT().Uninstall(gomock.Any(), gomock.Any())
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
				pvcList := list.(*corev1.PersistentVolumeClaimList)
				pvcList.Items = []corev1.PersistentVolumeClaim{
					{},
				}
			}).Return(errTest)
		err := r.Finalize(ctx, m)
		assert.Error(t, err)
	})

	t.Run("storage delete failed", func(t *testing.T) {
		defer env.checkMocks()
		mockHelm.EXPECT().Uninstall(gomock.Any(), gomock.Any())
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any())
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
				pvcList := list.(*corev1.PersistentVolumeClaimList)
				pvcList.Items = []corev1.PersistentVolumeClaim{
					{},
				}
			})
		mockClient.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errTest)
		err := r.Finalize(ctx, m)
		assert.Error(t, err)
	})

	t.Run("dependency external ignored", func(t *testing.T) {
		m.Spec.Dep.Etcd.External = true
		m.Spec.Dep.Pulsar.External = true
		m.Spec.Dep.Storage.External = true
		m.Spec.Dep.Etcd.InCluster = nil
		m.Spec.Dep.Pulsar.InCluster = nil
		m.Spec.Dep.Storage.InCluster = nil
		err := r.Finalize(ctx, m)
		assert.NoError(t, err)
	})

}

func TestCluster_SetDefaultStatus(t *testing.T) {
	env := newClusterTestEnv(t)
	defer env.checkMocks()
	r := env.Reconciler
	mockClient := env.MockClient
	ctx := env.ctx
	errTest := errors.New("test")

	// no status, set default failed
	m := env.Inst
	mockClient.EXPECT().Status().Return(mockClient)
	mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errTest)
	_, err := r.SetDefaultStatus(ctx, &m)
	assert.Error(t, err)

	// no status, set default ok
	m = env.Inst // ptr value changed, need reset
	mockClient.EXPECT().Status().Return(mockClient)
	mockClient.EXPECT().Update(gomock.Any(), gomock.Any())
	updated, err := r.SetDefaultStatus(ctx, &m)
	assert.NoError(t, err)
	assert.True(t, updated)

	// has status, not set
	m = env.Inst // ptr value changed, need reset
	m.Status.Status = v1alpha1.StatusCreating
	updated, err = r.SetDefaultStatus(ctx, &m)
	assert.NoError(t, err)
	assert.False(t, updated)
}

func TestCluster_ReconcileAll(t *testing.T) {
	env := newClusterTestEnv(t)
	defer env.checkMocks()
	r := env.Reconciler
	ctx := env.ctx
	m := env.Inst

	mockGroup := NewMockGroupRunner(env.Ctrl)
	defaultGroupRunner = mockGroup

	mockGroup.EXPECT().Run(gomock.Len(4), gomock.Any(), m)

	err := r.ReconcileAll(ctx, m)
	assert.NoError(t, err)
}

func TestCluster_ReconcileMilvus(t *testing.T) {
	env := newMilvusTestEnv(t)
	defer env.tearDown()
	r := env.Reconciler
	mockClient := env.MockClient
	ctx := env.ctx
	m := env.Inst

	// dep not ready
	err := r.ReconcileMilvus(ctx, m)
	assert.NoError(t, err)

	// dep ready
	mockGroup := NewMockGroupRunner(env.Ctrl)
	defaultGroupRunner = mockGroup

	m.Status.Conditions = []v1alpha1.MilvusCondition{
		{
			Type:   v1alpha1.EtcdReady,
			Status: corev1.ConditionTrue,
		},
		{
			Type:   v1alpha1.PulsarReady,
			Status: corev1.ConditionTrue,
		},
		{
			Type:   v1alpha1.StorageReady,
			Status: corev1.ConditionTrue,
		},
	}

	gomock.InOrder(
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "")),
		// get secret of minio
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "mockErr")),
		mockClient.EXPECT().
			Create(gomock.Any(), gomock.Any()).Return(nil),
		mockGroup.EXPECT().Run(gomock.Len(4), gomock.Any(), m),
	)

	err = r.ReconcileMilvus(ctx, m)
	assert.NoError(t, err)
}
