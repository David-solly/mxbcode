# Device UID Generator

Server implementation of the required functionality and stretch objectives.

# Api endpoints

### Requires the CLI to run in server mode

#### {URL}/generate/{reqid}

Supply any value for the reqid to keep your requests unique, use the same reqid to retrieve past responses.
This endpoint automatically generates 100 DevEUIs.

#### {URL}/view/{shortcode}

Retrieves the full DevEUI from a shortcode - if one exists on the system.

# LoraWan Endpoint

The endpoint should respond with a 200 for new registers and 422 if already registered, currently it only issues 200 response code regardless of the status. I've created an optional mockup registration server to use for testing purposes if needed that performs as needed with 200 and 422 as appropriate. it requires no arguments to run.

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

To run the CLI as a server - Supply a port number when running the app. eg - `go run . -port=8282` this will start the cli in server mode and expose the above generate endpoints.

## Cache

The cli includes an in-memory cache and has working bindings and tests for a Redis (expandable to other) database. The in-memory cache is cleared once the server is shutdown.
To run with a known redis instance, supply the redis address flag with the redis endpoint - eg `-redis-addr=192.168.99.100:6379` .
