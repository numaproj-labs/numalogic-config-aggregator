package leaderelection

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func Test_RunOrDie(t *testing.T) {
	// FIXME: DATA RACE
	k8sCli := k8sfake.NewSimpleClientset()
	elector1 := NewK8sLeaderElector(k8sCli, "test-ns", "my-lock", "e1", WithLeaseDuration(10*time.Second), WithRenewDeadline(5*time.Second))
	elector2 := NewK8sLeaderElector(k8sCli, "test-ns", "my-lock", "e2", WithLeaseDuration(10*time.Second), WithRenewDeadline(5*time.Second))

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	defer cancel1()
	defer cancel2()
	run1, run2 := false, false
	stop1, stop2 := false, false

	end1, end2 := false, false

	go func() {
		elector1.RunOrDie(ctx1, LeaderCallbacks{
			OnStartedLeading: func(_ context.Context) {
				fmt.Println("start 101")
				run1 = true
			},
			OnStoppedLeading: func() {
				fmt.Println("stop 101")
				stop1 = true
			},
		})
		end1 = true
	}()

	go func() {
		elector2.RunOrDie(ctx2, LeaderCallbacks{
			OnStartedLeading: func(_ context.Context) {
				fmt.Println("start 201")
				run2 = true
			},
			OnStoppedLeading: func() {
				fmt.Println("stop 201")
				stop2 = true
			},
		})
		end2 = true
	}()
	time.Sleep(3 * time.Second)
	assert.True(t, run1 || run2)
	assert.False(t, run1 && run2)
	if run1 {
		cancel1()
	}
	if run2 {
		cancel2()
	}
	time.Sleep(3 * time.Second)
	assert.True(t, stop1 || stop2)
	assert.False(t, stop1 && stop2)
	assert.True(t, end1 || end2)
	assert.False(t, end1 && end2)
}
