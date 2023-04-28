# Developer Documentation

## Testing CIDs

* `QmdpRDxYVnCdvqghj7KfzcaLyqo2NdHcXFriXo3Q7B9SsC` [link](https://gateway.pinata.cloud/ipfs/QmdpRDxYVnCdvqghj7KfzcaLyqo2NdHcXFriXo3Q7B9SsC/) -- the `testdata` directory as of 27/04/2023 -- [Example Result](http://amplify.bacalhau.org/#/queue/193bff74-81b6-4075-a99d-daff216e240b/show)

## Job Interface

Jobs are individual units of work that execute in a worker, which is just a 
simple goroutine. You can think of a job being a Bacalhau job, but they could be
anything. The crucial element here is that Amplify needs to chain jobs together
and so we need to define a common interface that all jobs must implement. We
have tried to keep this interface as generic as possible, but we must work
within the constraints of the Bacalhau API.

### Definition of a Job

> Note that the definition of a job is quite generic in the code, but for now
> we expect jobs to be 
> [Bacalhau Docker jobs](https://docs.bacalhau.org/getting-started/docker-workload-onboarding)
> , i.e. containers

* The job must be a Bacalhau-like job
* The job must conform to the [input](#job-inputs) and [output](#job-outputs)
  specifications.
* The job must be named and configured to run in the `config.yml` file
* Jobs must have a unique name

### Job Inputs

* All inputs are passed via the `/inputs` directory is mounted as a volume in
  the container
* Jobs must operate on every file and directory in the `/inputs` directory
  recursively
* Previous nodes may be skipped due to a predicate, so don't assume specific inputs will be present

### Job Outputs

* Derivative files must be written to the `/outputs` directory
* Derivative files should have names that are both unique and consistent with 
  the original file name
* Metadata must be written to `stdout` (so subsequent jobs can predicate)
* Errors must be written to `stderr`
* Jobs should refrain from breaking changes to the output directory

## Workflow Interface

Workflows are a collection of jobs that are chained together into a DAG. Amplify
workflows are defined in a YAML file, which is then parsed and executed by the
Amplify engine.

The interesting thing about Amplify workflows is that they run only when they
predicate the results of the previous job. This means that we can define a
workflow that only runs on specific types of data (images, for example).

### Definition of a Workflow

* Workflows must be configured in the `config.yml` file
* Workflows can be duplicated
* Workflows must have a unique name
* Given a single root CID, a composite CID will be generated with the results of
  all workflows

## Amplify Architecture

The image below shows a simplified version of the Amplify architecture.

![Amplify Architecture](./images/amplify_architecture.png)
