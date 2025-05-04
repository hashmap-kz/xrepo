package concur

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Task function for testing
func mockTaskW(_ context.Context, input int) (int, error) {
	if input%2 == 0 {
		return input * 2, nil // Double even numbers
	}
	return 0, errors.New("odd number error") // Return error for odd numbers
}

func TestProcessConcurrentlyWithResultAndLimit(t *testing.T) {
	tasks := []int{1, 2, 3, 4, 5, 6} // 2, 4, 6 will succeed
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, 2, tasks, mockTaskW, nil)
	assert.Len(t, results, 3)
	assert.Len(t, errs, 3) // 1, 3, 5 should fail
}

func TestProcessConcurrentlyWithResultAndLimit_Cancellation(t *testing.T) {
	tasks := []int{2, 4, 6, 8, 10} // All tasks should return valid results

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, 2, tasks, mockTaskW, nil)

	assert.Empty(t, results) // Should return no results
	assert.Empty(t, errs)    // Should return no errors since no task runs
}

func TestProcessConcurrentlyWithResultAndLimit_WorkerLimit(t *testing.T) {
	tasks := make([]int, 100)
	for i := 0; i < 100; i++ {
		tasks[i] = i
	}

	ctx := context.Background()
	start := time.Now()
	_, _ = ProcessConcurrentlyWithResultAndLimit(ctx, 5, tasks, func(_ context.Context, i int) (int, error) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		return i, nil
	}, nil)

	duration := time.Since(start)
	assert.Greater(t, duration, 200*time.Millisecond) // Should take more than 200ms (ensuring limited concurrency)
}

func TestProcessConcurrentlyWithResultAndLimit_LargeInput(t *testing.T) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}

	ctx := context.Background()
	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, 10, tasks, mockTaskW, nil)

	assert.Greater(t, len(results), 0)        // Ensure some results are returned
	assert.LessOrEqual(t, len(results), 5000) // At most half should be filtered
	assert.Len(t, errs, 5000)                 // Half should fail
}

// benchmarks

func BenchmarkProcessConcurrentlyWithResultAndLimit(b *testing.B) {
	tasks := make([]int, 10000)
	for i := 0; i < len(tasks); i++ {
		tasks[i] = i
	}
	ctx := context.Background()

	taskFunc := func(_ context.Context, input int) (int, error) {
		if input%2 == 0 {
			return input * 2, nil // Double even numbers
		}
		return 0, errors.New("odd number error") // Return error for odd numbers
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessConcurrentlyWithResultAndLimit(ctx, 10, tasks, taskFunc, nil)
	}
}

// v2

func TestProcessConcurrentlyWithResultAndLimit_Success(t *testing.T) {
	t.Parallel()

	tasks := []int{1, 2, 3, 4, 5}
	workerLimit := 3
	taskFunc := func(_ context.Context, task int) (int, error) {
		return task * 2, nil
	}
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, workerLimit, tasks, taskFunc, nil)
	assert.Empty(t, errs, "expected no errors")

	// Since each task's result is stored by its index,
	// the order of results should match the order of tasks.
	expected := []int{2, 4, 6, 8, 10}
	assert.Equal(t, expected, results, "results should match expected values")
}

func TestProcessConcurrentlyWithResultAndLimit_Error(t *testing.T) {
	t.Parallel()

	tasks := []int{1, 2, 3}
	workerLimit := 2
	taskFunc := func(_ context.Context, task int) (int, error) {
		return 0, fmt.Errorf("error on task %d", task)
	}
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, workerLimit, tasks, taskFunc, nil)
	assert.Empty(t, results, "expected no results")
	assert.Len(t, errs, len(tasks), "expected an error per task")

	// Verify error messages.
	for i, err := range errs {
		expectedErrMsg := fmt.Sprintf("error on task %d", tasks[i])
		assert.EqualError(t, err, expectedErrMsg)
	}
}

func TestProcessConcurrentlyWithResultAndLimit_Mixed(t *testing.T) {
	t.Parallel()

	tasks := []int{1, 2, 3, 4}
	workerLimit := 2
	taskFunc := func(_ context.Context, task int) (int, error) {
		// Return an error for even tasks.
		if task%2 == 0 {
			return 0, fmt.Errorf("error on task %d", task)
		}
		return task * 10, nil
	}
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, workerLimit, tasks, taskFunc, nil)
	// Expect results for tasks 1 and 3; errors for tasks 2 and 4.
	assert.Len(t, results, 2, "expected two results")
	assert.Len(t, errs, 2, "expected two errors")

	// Verify the results.
	expectedResults := []int{10, 30}
	// The order is preserved by the index.
	assert.Equal(t, expectedResults, results, "results should match expected values")
}

func TestProcessConcurrentlyWithResultAndLimit_ContextCancellation(t *testing.T) {
	t.Parallel()

	tasks := []int{1, 2, 3, 4, 5}
	workerLimit := 2
	var mu sync.Mutex
	executedTasks := make([]int, 0)
	taskFunc := func(_ context.Context, task int) (int, error) {
		// Simulate work.
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		executedTasks = append(executedTasks, task)
		mu.Unlock()
		return task * 3, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel context after 150ms.
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, workerLimit, tasks, taskFunc, nil)
	// We cannot guarantee how many tasks complete due to cancellation.
	totalOutcomes := len(results) + len(errs)
	assert.Less(t, totalOutcomes, len(tasks)+1, "not all tasks should complete after cancellation")
	assert.NotNil(t, ctx.Err(), "expected context to be canceled")
}

func TestProcessConcurrentlyWithResultAndLimit_ConcurrencySafety(t *testing.T) {
	t.Parallel()

	numTasks := 1000
	tasks := make([]int, numTasks)
	for i := 0; i < numTasks; i++ {
		tasks[i] = i
	}
	workerLimit := 50
	taskFunc := func(_ context.Context, task int) (int, error) {
		return task * 2, nil
	}
	ctx := context.Background()

	results, errs := ProcessConcurrentlyWithResultAndLimit(ctx, workerLimit, tasks, taskFunc, nil)
	assert.Empty(t, errs, "expected no errors")
	assert.Len(t, results, numTasks, "expected all tasks to be processed")
	// Verify each result.
	for i, task := range tasks {
		expected := task * 2
		assert.Equal(t, expected, results[i], "result for task %d should be %d", task, expected)
	}
}
