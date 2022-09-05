`ringbuffer` is a simple, fast, and thread-safe ring buffer implementation for `GO` that is not limited to a single
type(usually Byte) as other implementations.
It relies on GO's recent generics support, so it requires go 1.18 or higher.

## Usage

```go
package main

import (
	"fmt"
	"github.com/IllicLanthresh/ringbuffer"
)

type Person struct {
	Name string
	Age  int
}

func main() {
	// Create a new ring buffer of type Person with a capacity of 10 and a push buffer size of 10 too.
	// The capacity is the maximum number of elements that can be stored in the buffer.
	// The ring buffer performs operations in a dedicated goroutine sequentially.
	// Those operations are buffered and the pushBufferSize is the maximum number of operations that can be buffered.
	rb := ringbuffer.New[Person](10, 10)

	// Add some elements, adding to a total count of 4 elements for now...
	// We will use PushAndWait instead of Push to ensure that we don't run into race conditions during the example
	// Push does not block execution, since the ring pusher is running on a separate goroutine
	rb.PushAndWait(&Person{Name: "John", Age: 20})
	rb.PushAndWait(
		&Person{Name: "Jane", Age: 21},
		&Person{Name: "Jack", Age: 22},
		&Person{Name: "Jill", Age: 23},
	)

	// Pop the oldest element, leaving a total count of 3 elements in the buffer, of a total capacity of 10
	first, ok := rb.Pop()
	if ok {
		fmt.Println(first) // Person{Name: "John", Age: 20}
	}

	// The oldest element is now the second element we pushed in (Jane), 
	// add some more elements to the buffer, exceeding the capacity of 10
	rb.PushAndWait(
		&Person{Name: "Jenny", Age: 24},
		&Person{Name: "Adam", Age: 25},
		&Person{Name: "Eve", Age: 26},
		&Person{Name: "Bob", Age: 27},
		&Person{Name: "Alice", Age: 28},
		&Person{Name: "Charlie", Age: 29},
		&Person{Name: "Diana", Age: 30},
	)

	// The buffer now contains 10 elements, the oldest being Jane, and the newest being Diana,
	// so adding more elements will cause the oldest elements to be overwritten
	rb.PushAndWait(&Person{Name: "Ella", Age: 31})

	// The buffer now still contains 10 elements, the oldest being Jack, and the newest being Ella,
	// Jane has now been overwritten and is no longer in the buffer

	// Note here the use of Push instead of PushAndWait
	rb.Push(&Person{Name: "Frank", Age: 32})

	// The buffer will now still contain 10 elements, the oldest being Jill, and the newest being Frank,
	// Jack has now been overwritten and is no longer in the buffer, but since we used Push instead of PushAndWait,
	// the ring pusher goroutine may not have had time to process the new element yet, so we will wait for it to do so
	rb.Wait()

	// You can also iterate over the buffer, starting from the oldest element,
	// this will not remove any elements from the buffer
	for _, person := range rb.Iterate() {
		fmt.Println(*person)
	}

	persons := rb.Flush() // Flush the buffer, returning all elements in the buffer and emptying it, oldest to newest
	for _, person := range persons {
		fmt.Println(*person)
	}

	// AddQueueIdleHook allows you to add a function that will be called when the ring buffer is idle
	// This is useful if you want to perform some action when the buffer is at a certain size, when it is empty,
	// when it is full, when it is no longer full... You get the idea.
	rb.AddQueueIdleHook(func() {
		if rb.Len() == 0 {
			fmt.Println("The buffer is empty!")
		}
	})

	// It is recommended to close the ring buffer when you are done with it,
	// this will stop the ring pusher goroutine and free up resources
	rb.Close()

	// You can also have the channel close after a flush
	rb.FlushAndClose() // This will panic as the buffer is already closed, otherwise it would return the same as Flush()
}
```

## TODO
The project is still missing tests, benchmarks, and documentation.

## License
MIT
