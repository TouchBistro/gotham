// Package circleci provides an HTTP client for the CircleCI API (v1.1 and v2).
//
// It enables callers to query projects, manage pipelines and workflows, and
// follow or unfollow projects on CircleCI. A builder-style constructor is used
// for configuration and future extensibility.
//
// Basic usage:
//
//	c := circleci.NewClientBuilder().
//	    WithToken("my-api-token").
//	    Build()
//	project, err := c.GetProject(ctx, "TouchBistro", "my-service")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(project.Name)
package circleci
