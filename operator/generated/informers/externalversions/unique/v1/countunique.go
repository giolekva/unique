// gen

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	uniquev1 "github.com/giolekva/unique/operator/apis/unique/v1"
	versioned "github.com/giolekva/unique/operator/generated/clientset/versioned"
	internalinterfaces "github.com/giolekva/unique/operator/generated/informers/externalversions/internalinterfaces"
	v1 "github.com/giolekva/unique/operator/generated/listers/unique/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// CountUniqueInformer provides access to a shared informer and lister for
// CountUniques.
type CountUniqueInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.CountUniqueLister
}

type countUniqueInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewCountUniqueInformer constructs a new informer for CountUnique type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewCountUniqueInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredCountUniqueInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredCountUniqueInformer constructs a new informer for CountUnique type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredCountUniqueInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.LekvaV1().CountUniques(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.LekvaV1().CountUniques(namespace).Watch(context.TODO(), options)
			},
		},
		&uniquev1.CountUnique{},
		resyncPeriod,
		indexers,
	)
}

func (f *countUniqueInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredCountUniqueInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *countUniqueInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&uniquev1.CountUnique{}, f.defaultInformer)
}

func (f *countUniqueInformer) Lister() v1.CountUniqueLister {
	return v1.NewCountUniqueLister(f.Informer().GetIndexer())
}
