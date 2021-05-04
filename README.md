# GoDotEnv ![CI](https://github.com/alois9866/godotenv/workflows/CI/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/alois9866/godotenv)](https://goreportcard.com/report/github.com/alois9866/godotenv)

This is a fork of [godotenv](https://github.com/joho/godotenv). It rethinks some points of the original idea, narrowing
the possible use scenarios. The main purpose of the change is to be able to read not only dotenv variables, **but also
system variables**.

This project contains only the functions that I consider to be essential. If you need a more feature-rich solution,
please check out the original repository.

## Installation

```shell
go get github.com/alois9866/godotenv
```

## Usage

Add your application configuration to your `.env` file in the root of your project:

```shell
S3_BUCKET=YOURS3BUCKET
SECRET_KEY=YOURSECRETKEYGOESHERE
```

You can also set some variables in your shell or when running you app as `ANOTHER_KEY=KEYGOESHERE ./app`

Then in your Go app you can do something like

```go
package main

import (
    "github.com/alois9866/godotenv"
)

func main() {
    env, _ := godotenv.Variables().Get()

    s3Bucket := env["S3_BUCKET"]
    secretKey := env["SECRET_KEY"]
    anotherKey := env["ANOTHER_KEY"]

    // ...
}
```

Basically if you want to read all environment variables, you can call:

```go
env, _ := godotenv.Variables().Get()
````

By default, dotenv variables will take precedence over system variables. If you want to use values from system
environment over values from dotenv files, you can use this:

```go
env, _ := godotenv.Variables().PrioritizeSystem().Get()
````

If you want to check that some specific variables are available, you can call:

```go
env, notFound := godotenv.Variables("ENV_VAR1", "ENV_VAR2").Get()
```

In this case, if some of those variables are not set, `notFound` will contain their names.

If you want to use files other than `.env`, you can do that too:

```go
env, notFound := godotenv.Variables("ENV_VAR1", "ENV_VAR2").GetFrom("file1", "file2")
```

### File formatting

If you want to be really fancy with your env file you can do comments and exports (below is a valid env file):

```shell
# I am a comment and that is OK
SOME_VAR=someval
SOME_VAR2="someval2"
FOO=BAR # comments at line end are OK too
export BAR=BAZ
```

Or finally you can do YAML(ish) style:

```yaml
FOO: bar
BAR: baz
```

If you want to know more about original dotenv usage convention, you can read about
it [here](https://github.com/bkeepers/dotenv#what-other-env-files-can-i-use).

## Who?

The original Ruby library [dotenv](https://github.com/bkeepers/dotenv) was written
by [Brandon Keepers](http://opensoul.org/).

The original Go library was written by [John Barton](https://johnbarton.co/) based off the tests/fixtures in the
original library.

This version is written by [me](https://github.com/alois9866).
