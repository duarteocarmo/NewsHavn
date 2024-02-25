# hyggenews

## Prerequisites

* [Go](https://go.dev/)
* [Just](https://github.com/casey/just)

## Run locally

1. Make sure you have set up the env variables (see `.env_example`)
2. Start DB
```bash
$ just clean-db
```
3. Run
```bash
$ just run
```

The app should spin up on `localhost:8080`. 

For more options, run `just -l`
