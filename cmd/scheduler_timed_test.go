package cmd

import (
	"context"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("getNextRunTime", func() {
	It("should return a time in the future for a time ahead of now", func() {
		// 1 minute from now (HH:MM format, so truncated to the minute)
		future := time.Now().Add(2 * time.Minute).Format("15:04")
		next := getNextRunTime([]string{future})
		Expect(next.After(time.Now())).To(BeTrue())
	})

	It("should add 24h when all times are in the past", func() {
		// 2 minutes ago
		past := time.Now().Add(-2 * time.Minute).Format("15:04")
		next := getNextRunTime([]string{past})
		// next run should be ~22+ hours from now (tomorrow)
		Expect(next.After(time.Now().Add(20 * time.Hour))).To(BeTrue())
	})

	It("should pick the earliest future time from multiple times", func() {
		sooner := time.Now().Add(2 * time.Minute).Format("15:04")
		later := time.Now().Add(30 * time.Minute).Format("15:04")
		next := getNextRunTime([]string{later, sooner})
		expected := getNextRunTime([]string{sooner})
		// Both should resolve to the same next run (the sooner one)
		Expect(next.Equal(expected)).To(BeTrue())
	})

	It("should return a non-zero time for any non-empty input", func() {
		anyTime := "12:00"
		next := getNextRunTime([]string{anyTime})
		Expect(next.IsZero()).To(BeFalse())
	})
})

var _ = Describe("runTimedTask", func() {
	It("should exit promptly when context is cancelled before the scheduled time", func() {
		ctx, cancel := context.WithCancel(context.Background())
		queue := make(chan config.ScheduledTask, 1)

		// Schedule far in the future (next occurrence of 00:01, essentially ~24h away at worst)
		task := config.ScheduledTask{
			Name:     "far-future-task",
			Commands: []string{"add-routes"},
			Configs:  []string{"/cfg.yaml"},
			Times:    []string{"00:01"},
		}

		done := make(chan struct{})
		go func() {
			runTimedTask(ctx, task, queue)
			close(done)
		}()

		cancel()

		Eventually(done, "500ms").Should(BeClosed())
		Expect(queue).To(HaveLen(0))
	})

	It("should fire immediately when next run time is effectively now (≤1 min away)", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		queue := make(chan config.ScheduledTask, 2)

		// Use a time 1 minute in the future — getNextRunTime will produce a wait < 65s.
		// We test that the goroutine sends to the queue, then cancel to stop repeat.
		nextMin := time.Now().Add(1 * time.Minute)
		timeStr := nextMin.Format("15:04")

		task := config.ScheduledTask{
			Name:     "next-minute-task",
			Commands: []string{"add-routes"},
			Configs:  []string{"/cfg.yaml"},
			Times:    []string{timeStr},
		}

		go func() {
			runTimedTask(ctx, task, queue)
		}()

		var received config.ScheduledTask
		Eventually(queue, "70s", "100ms").Should(Receive(&received))
		Expect(received.Name).To(Equal("next-minute-task"))
		cancel()
	})
})

var _ = Describe("runTask", func() {
	It("should delegate to runIntervalTask when Interval is set", func() {
		ctx, cancel := context.WithCancel(context.Background())
		queue := make(chan config.ScheduledTask, 1)

		task := config.ScheduledTask{
			Name:     "interval-dispatch",
			Commands: []string{"add-routes"},
			Configs:  []string{"/cfg.yaml"},
			Interval: "10ms",
		}

		go func() {
			runTask(ctx, task, queue)
		}()

		var received config.ScheduledTask
		Eventually(queue, "2s", "5ms").Should(Receive(&received))
		Expect(received.Name).To(Equal("interval-dispatch"))

		cancel()
	})

	It("should delegate to runTimedTask when Times is set (ctx cancel test)", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel before start

		queue := make(chan config.ScheduledTask)
		task := config.ScheduledTask{
			Name:     "cancelled-timed",
			Commands: []string{"add-routes"},
			Configs:  []string{"/cfg.yaml"},
			Times:    []string{"23:59"},
		}

		done := make(chan struct{})
		go func() {
			runTask(ctx, task, queue)
			close(done)
		}()

		Eventually(done, "500ms").Should(BeClosed())
	})

	It("should exit immediately from runIntervalTask path when ctx is cancelled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		queue := make(chan config.ScheduledTask)
		task := config.ScheduledTask{
			Name:     "cancelled-interval",
			Commands: []string{"add-routes"},
			Configs:  []string{"/cfg.yaml"},
			Interval: "1h",
		}

		done := make(chan struct{})
		go func() {
			runTask(ctx, task, queue)
			close(done)
		}()

		Eventually(done, "500ms").Should(BeClosed())
	})
})
