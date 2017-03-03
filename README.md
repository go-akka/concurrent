CONCURRENT
==========

`example.go`

```go
package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-akka/concurrent"
)

func main() {
	pool := concurrent.NewFixedRoutinePool(10, 100)

	taskGenerator := func(n int) func() (int, error) {
		return func() (int, error) {
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(500)))
			return n * 2, nil
		}
	}

	var tasks []interface{}
	for i := 1; i <= 50; i++ {
		tasks = append(tasks, taskGenerator(i))
	}

	futures, err := pool.InvokeAll(tasks)
	if err != nil {
		fmt.Println(err)
		return
	}

	for i := 0; i < len(futures); i++ {
		if futures[i].IsDone() {
			futures[i].Get().V(func(n int) { fmt.Printf("%d\t", n) })
		}
	}
}
```

```bash
> go run example.go
2	4	6	8	10	12	14	16	18	20	22
24	26	28	30	32	34	36	38	40	42	44
46	48	50	52	54	56	58	60	62	64	66
68	70	72	74	76	78	80	82	84	86	88
90	92	94	96	98	100
```

The argument of `futures[i].Get().V(fn)` is the return value of `func() (int, error)`, so here should be `func(n int)` or `func(n int, e error)`.


`example.go`

```go
package main

import (
	"fmt"

	"github.com/go-akka/concurrent"
	_ "github.com/go-akka/concurrent/global"
)

func main() {

	future := concurrent.NewFutureTask(
		func() (int, error) {
			return 0, fmt.Errorf("error from future")
		})

	future.OnComplete(
		func(v int) error {
			fmt.Println("OnComplete 1:", v)
			return nil
		})

	future.OnComplete(
		func(v int, err error) {
			fmt.Println("OnComplete 2:", err)
			return
		})

	future.OnComplete(
		func(v int, err error) error {
			fmt.Println("OnComplete 3: I will report error to ExecutionContext")
			return fmt.Errorf("error from OnComplete 3, original error is: %s", err.Error())
		})

	f := future.
		AndThen(
			func(v int) {
				fmt.Printf("AndThen1: I do not care error\n")
			}).
		AndThen(func(v int, err error) {
			if err != nil {
				fmt.Printf("AndThen2: %s\n", err)
				return
			}

			fmt.Printf("AndThen2", v)
		}).
		AndThen(func(v int) error {
			fmt.Println("AndThen3: I will report error to ExecutionContext")
			return fmt.Errorf("error from AndThen3")
		})

	f.Get()
}

```

```bash
> go run example.go
OnComplete 1: 0
OnComplete 2: error from future
OnComplete 3: I will report error to ExecutionContext
AndThen1: I do not care error
AndThen2: error from future
AndThen3: I will report error to ExecutionContext
```

