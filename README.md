# Device UID Generator

Server implementation of the required functionality and stretch objectives.

# Api endpoints

#### {URL}/generate/{count}/{uid}

Generates the DevEui list with a count and unique id- ID can be anything.
A cache will be hit if the count and uid are equal and from the request orinates from the same host. Varying any one of these three parameters will result in a newly generated and registered set.

#### {URL}/view/{shortcode}

Retrieves the full DevEUI from a shortcode - if one exists on the system.

#### {URL}/activate/{shortcode}

Activates a shortcode with the provider directly.

# Testing

To Run the server level tests - the registration endpoint needs to be accessible (registration url) - a sample web client is included that testing can be carried out against. This client should run on port 8080 taking no arguments. - located at `cmd/client/main.go`

### Clearing the sample endpoint memory

To clear the mock registration endpoint - send a blank 'Get' request to the base path. if on local host `http://localhost:8080/`

## Api Testing

To carry out API Endpoint tests situated in `cmd/server/router_test.go` - requires the main server to be running on port 8082, located at `cmd/server/main.go`

### Running the cli

The cli does not require any arguments to generate a batch of 100, that is the default beahiour. There are several flags which are available to alter the behaviour;

`-count` a custom amount to generate - limited to a max of 100 as per the spec. Attempting any higher numbers will result in an error.

`-l` flag to set the last shortcode number of a batch to increment from. This is needed only if running the cli in application mode to avoid regenerating the same batch.
If running in server mode, the increment operation is kept in the appropriate cache and read from there. This can still be used to override the starting value.

The value will reset to 00000 upon termination unless backed by a redis cache.

`-port` the only flag required to transform the cli to a server. The port to bind the server to

`-redis-addr` the redis address to bind to. Leaving this blank will automatically switch to the in-memory cache.

`-reg-url` takes a fully qualified device registration endpoint, if none provided - defaults to the endpoint provided in the spec

### Running the server

To run the server - Supply a port number and an optional bind address. This will override the `-count` argument if used together.
`go run ./cmd/server/ -port=8082` - optional `-addr=my.bind.address`.

## Cache

The cli includes an in-memory cache and has working bindings and tests for a Redis (expandable to other) database. The in-memory cache is cleared once the server is shutdown.
To run with a known redis instance, supply the redis address flag with the redis endpoint - eg `-redis-addr=192.168.99.100:6379` .

## Building the image

Ensure go mod vendor is called to cache all dependencies locally. The build won't run otherwise.
Removes the need to supply credentials for git/docker hub

`MMAX\barcode-system$docker build -t mmax-bcode:latest . -f ./dockerfile.server`

The Multi-stage docker file builds on a minimal alpine image so it's small approx ~13MB - however has no shell environment.
