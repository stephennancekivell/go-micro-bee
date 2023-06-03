# Micro Bee

> Busy as a Bee: Turbocharge Your Workflows with Go Micro Bee's Micro Batching

Micro Bee is a small library to do micro batch processing in golang.

Mico batching is the process of grouping Jobs together to process them at once. This can bring significant performance benefits when processing a large amounts of data you need want to minimise the about of processes you need. Eg minimise database transactions.

● it should allow the caller to submit a single Job, and it should return a JobResult
● it should process accepted Jobs in batches using a BatchProcessor
○ Don't implement BatchProcessor. This should be a dependency of your library.
● it should provide a way to configure the batching behaviour i.e. size and frequency
● it should expose a shutdown method which returns after all previously accepted Jobs are
processed

```go
type BatchProcessor[A,B] = func (jobs []Job[A]) []JobResult[A,B]

processBatch := func(jobs []Job[String]) ([]JobResult[string,int],error) {
    queryArgs := make([]string, len(jobs))
    for i, job := range jobs {
        queryArgs[i] = "tag like '%"+job+"%'"
    }

    results, err := store.Lookup(queryArgs)
    if err != nil {
        return nil,err
    }

    results := make([]JobResult[string,int], len(jobs))
    for i,job := range jobs {
        result,ok = results[job]
        if !ok {
            log.Warn("no result for job defaulting to zero: %v",job)
            results[i] = JobResult{Job: job, result: 0}
        } else {
            results[i] = JobResult{Job: job, result: result}
        }
    }
}

processor := microbee.NewProcessor(processBatch, 10, time.second)

jobResult1 := processor.Submit(Job{ Value:"hello" })
jobResult2 := processor.Submit(Job{ Value:"World" })

result1, err := jobResutl1.Get()
if err != nil {
    panic(err)
}
result2, err := jobResutl2.Get()
if err != nil {
    panic(err)
}

processor.Cancel()
processor.Stop()
```
