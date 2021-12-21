package kubeutils

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	"hieu.le/secrets-sync/internal/pkg/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// KubeClient wrap the k8s clientset
type KubeClient struct {
	clientset *kubernetes.Clientset
}

// InitKube initializes the k8s clientset
func InitKube() (*KubeClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.WithFields(log.Fields{}).Error("Can not initialize in-cluster config: ", err)
		return nil, err
	}
	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.WithFields(log.Fields{}).Error("Can not create kubeclient: ", err)
		return nil, err
	}
	cs := KubeClient{
		clientset: c,
	}
	return &cs, nil
}

// StartWatcher starts watching the secret and update it accordingly
func (c *KubeClient) StartWatcher(ctx context.Context, wg *sync.WaitGroup, s utils.Secret, id int) {
	defer wg.Done()
	op := metav1.ListOptions{
		FieldSelector: "metadata.name=" + s.Name,
	}
	factory := informers.NewSharedInformerFactoryWithOptions(
		c.clientset,
		0,
		informers.WithNamespace(s.SourceNamespace),
		informers.WithTweakListOptions(op.DeepCopyInto))
	informer := factory.Core().V1().Secrets().Informer()
	stopper := make(chan struct{})
	defer close(stopper)
	defer runtime.HandleCrash()

	mux := &sync.RWMutex{}
	synced := false
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(interface{}) {
			mux.RLock()
			defer mux.RUnlock()
			if !synced {
				return
			}
			log.WithFields(log.Fields{}).Info("Secret ", s.Name, " has been created.")
			c.createSecrets(s)
		},
		UpdateFunc: func(oldSec interface{}, newSec interface{}) {
			mux.RLock()
			defer mux.RUnlock()
			if !synced {
				return
			}
			if oldSec == newSec {
				log.WithFields(log.Fields{}).Debug("Secret has NOT been changed.")
			} else {
				log.WithFields(log.Fields{}).Info("Secret ", s.Name, " has been changed.")
				c.updateSecrets(s)
			}
		},
		DeleteFunc: func(interface{}) {
			mux.RLock()
			defer mux.RUnlock()
			if !synced {
				return
			}
			log.WithFields(log.Fields{}).Info("Secret ", s.Name, " has been deleted.")
			c.deleteSecrets(s)
		},
	})
	go informer.Run(stopper)

	isSynced := cache.WaitForCacheSync(stopper, informer.HasSynced)
	mux.Lock()
	synced = isSynced
	mux.Unlock()
	if !isSynced {
		log.WithFields(log.Fields{"context": "kubernetes"}).Info("Timed out waiting for caches to sync")
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}
	<-ctx.Done()
	c.deleteSecrets(s)
	log.WithFields(log.Fields{}).Info("Secret ", s.Name, " has been deleted.")
	log.WithFields(log.Fields{"worker": id}).Info("Stopped")
}

// SyncSecrets does an initial sync for all secrets
func (c *KubeClient) SyncSecrets(s utils.Secret) {
	sourceSecret, err := c.clientset.CoreV1().Secrets(s.SourceNamespace).Get(context.Background(), s.Name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{"secretName": s.Name}).Error("Error getting secret ", err)
		return
	}
	for _, n := range s.TargetNamespaces {
		targetSecret := c.makeSecret(sourceSecret, n)
		ok, err := c.secretExist(s.Name, n)
		if err != nil {
			return
		}
		if ok {
			c.updateSecret(targetSecret)
		} else {
			c.createSecret(targetSecret)
		}
	}
	log.WithFields(log.Fields{}).Info("Secret ", s.Name, " is synced in all namespaces")
}

// makeSecret copies a secret object and replace the target namespace
func (c *KubeClient) makeSecret(source *v1.Secret, namespace string) *v1.Secret {
	targetSecret := v1.Secret{
		Data:       source.Data,
		Type:       source.Type,
		StringData: source.StringData,
		ObjectMeta: metav1.ObjectMeta{
			Name:                       source.ObjectMeta.Name,
			Namespace:                  namespace,
			GenerateName:               source.GenerateName,
			Labels:                     source.Labels,
			Annotations:                source.Annotations,
			Finalizers:                 source.Finalizers,
			ClusterName:                source.ClusterName,
			ManagedFields:              source.ManagedFields,
			DeletionGracePeriodSeconds: source.DeletionGracePeriodSeconds,
		},
		TypeMeta:  source.TypeMeta,
		Immutable: source.Immutable,
	}
	return &targetSecret
}

// createSecret creates the secret in a namespace
func (c *KubeClient) createSecret(s *v1.Secret) error {
	_, err := c.clientset.CoreV1().Secrets(s.ObjectMeta.Namespace).Create(context.Background(), s, metav1.CreateOptions{})
	if err != nil {
		log.WithFields(log.Fields{"secretName": s.ObjectMeta.Name}).Error("Error creating secret ", err)
		return err
	}
	return nil
}

// updateSecret updates the secret in a namespace
func (c *KubeClient) updateSecret(s *v1.Secret) error {
	_, err := c.clientset.CoreV1().Secrets(s.ObjectMeta.Namespace).Update(context.Background(), s, metav1.UpdateOptions{})
	if err != nil {
		log.WithFields(log.Fields{"secretName": s.ObjectMeta.Name}).Error("Error updating secret ", err)
		return err
	}
	return nil
}

// createSecrets creates secrets in all target namespaces
func (c *KubeClient) createSecrets(s utils.Secret) {
	sourceSecret, err := c.clientset.CoreV1().Secrets(s.SourceNamespace).Get(context.Background(), s.Name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{"secretName": s.Name}).Error("Error getting secret ", err)
		return
	}
	for _, n := range s.TargetNamespaces {
		targetSecret := c.makeSecret(sourceSecret, n)
		c.createSecret(targetSecret)
	}
	log.WithFields(log.Fields{}).Info("Secret ", s.Name, " is created in all namespaces")
}

// updateSecrets updates secrets in all target namespaces
func (c *KubeClient) updateSecrets(s utils.Secret) {
	sourceSecret, err := c.clientset.CoreV1().Secrets(s.SourceNamespace).Get(context.Background(), s.Name, metav1.GetOptions{})
	if err != nil {
		log.WithFields(log.Fields{"secretName": s.Name}).Error("Error getting secret ", err)
		return
	}
	for _, n := range s.TargetNamespaces {
		targetSecret := c.makeSecret(sourceSecret, n)
		c.updateSecret(targetSecret)
	}
	log.WithFields(log.Fields{}).Info("Secret ", s.Name, " is updated in all namespaces")
}

// deleteSecrets deletes secrets in all target namespaces
func (c *KubeClient) deleteSecrets(s utils.Secret) {
	for _, n := range s.TargetNamespaces {
		err := c.clientset.CoreV1().Secrets(n).Delete(context.Background(), s.Name, metav1.DeleteOptions{})
		if err != nil {
			log.WithFields(log.Fields{"secretName": s.Name}).Error("Error creating secret ", err)
		}
	}
	log.WithFields(log.Fields{}).Info("Secret ", s.Name, " is removed in all namespaces")
}

// secretExist checks if a secret exists in a target namespace
func (c *KubeClient) secretExist(secretName string, namespace string) (bool, error) {
	l, err := c.clientset.CoreV1().Secrets(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.WithFields(log.Fields{"secretName": secretName}).Error("Error getting secret list ", err)
		return false, err
	}
	for _, s := range l.Items {
		if secretName == s.ObjectMeta.Name {
			return true, nil
		}
	}
	return false, nil
}
