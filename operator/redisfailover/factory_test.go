package redisfailover_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	mK8SService "github.com/spotahome/redis-operator/mocks/service/k8s"
	rfOperator "github.com/spotahome/redis-operator/operator/redisfailover"
)

// TestRetrieverWatchPropagatesErrorWithoutWrapping is a regression test for the
// operator crashing with a nil pointer dereference when the RedisFailover watch
// could not be established.
//
// The retriever's WatchFunc used to call watch.Filter unconditionally on the
// result of WatchRedisFailovers. On a watch error the typed client returns a
// nil watch.Interface, and watch.Filter immediately spawns a goroutine whose
// loop dereferences the source watcher's ResultChan, panicking the whole
// operator process (SIGSEGV) instead of letting the reflector retry the watch.
//
// The fix returns (nil, err) before wrapping. This test asserts the error is
// propagated and that no filtered watcher is returned (which is what would
// otherwise carry the panicking goroutine).
func TestRetrieverWatchPropagatesErrorWithoutWrapping(t *testing.T) {
	assert := assert.New(t)

	watchErr := errors.New("the server could not establish the watch")
	ms := &mK8SService.Services{}
	ms.On("WatchRedisFailovers", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, watchErr)

	retriever := rfOperator.NewRedisFailoverRetriever(
		rfOperator.Config{SupportedNamespacesRegex: ".*"},
		ms,
	)

	var (
		w   watch.Interface
		err error
	)
	assert.NotPanics(func() {
		w, err = retriever.Watch(context.Background(), metav1.ListOptions{})
	})
	assert.Equal(watchErr, err)
	assert.Nil(w, "a nil watcher must not be wrapped by watch.Filter")
	ms.AssertExpectations(t)
}

// TestRetrieverWatchWrapsWatcherOnSuccess verifies the happy path still wraps
// the underlying watcher (so namespace filtering stays in effect).
func TestRetrieverWatchWrapsWatcherOnSuccess(t *testing.T) {
	assert := assert.New(t)

	fake := watch.NewFake()
	defer fake.Stop()
	ms := &mK8SService.Services{}
	ms.On("WatchRedisFailovers", mock.Anything, mock.Anything, mock.Anything).
		Return(fake, nil)

	retriever := rfOperator.NewRedisFailoverRetriever(
		rfOperator.Config{SupportedNamespacesRegex: ".*"},
		ms,
	)

	w, err := retriever.Watch(context.Background(), metav1.ListOptions{})
	assert.NoError(err)
	assert.NotNil(w)
	ms.AssertExpectations(t)
}
