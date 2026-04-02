// Package shipit provides an HTTP client for the Shipit deployment engine API.
//
// It enables callers to list stacks, lock and unlock individual stacks, and
// perform bulk lock/unlock operations across all stacks with Go concurrency.
//
// Basic usage:
//
//	c := shipit.NewClient("https://shipit.example.com", "s3cr3t-password")
//	stacks, err := c.ListAllStacks(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, s := range stacks {
//	    fmt.Println(s.StackID())
//	}
package shipit
