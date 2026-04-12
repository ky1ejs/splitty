# Rules for assesing dependencies

## Things that MUST be considered when assessing dependencies

### 1. Size of the problem and the code needed to solve it

If the code needed is small, we should just write it ourselves. If the code needed is large, we should consider using a dependency.

### 2. Activity

How active is the dependency? Does it have recent commits? in the last 6 months? Does it have recent issues and pull requests? Are issues and PRs being addressed in a timely manner?

If so, this is a good sign that the dependency is being maintained and that any bugs or security issues will be addressed in a timely manner.

### 3. Popularity

How popular is the dependency? Does it have a large number of stars on GitHub? Does it have a large number of downloads (if this data is available)?

## When adding a dependency

We should ALWAYS use the latest stable version of the dependency. We should NEVER use a beta or alpha version of a dependency unless there is a very good reason to do so.