# Micro Bee

> Busy as a Bee: Turbocharge Your Workflows with Go Micro Bee's Micro Batching

Micro Bee is a small library to do micro batch processing in golang.

Mico batching is the process of grouping Jobs together to process them at once. This
can bring significant performance benefits when processing a large amounts of data. For
example, you could use micro batching to keep CPU cores busy or minimise the amount of
database transactions.

## Usage

In

```go
func MyInsertUsers(users []MyUser) []bool {
    ???
}

processor := microbee.NewProcessor(
    MyInsertUsers,
    100, // batch size
    100 * time.Millisecond // linger time
)

jobResult1 := processor.Submit( MyUser{ "foo" } )
jobResult2 := processor.Submit( MyUser{ "bar" } )

result1, err := jobResutl1.Get()
if err != nil {
    log.Print("error: unable to insert user")
}
result2, err := jobResutl2.Get()
if err != nil {
    log.Print("error: unable to insert user")
}

processor.Shutdown()
```

## testing

Tests can be run with `go test ./...`
