## DepsManager Backend

This service exposes an API for the UI.  
Refer to the Swagger(docs/swagger) documentation for more information about the available endpoints.  
It fetches npm dependencies from the deps.dev API and stores them in a SQLite database along with their SSF score.

To run unit tests, simply run `make test`.  
To run end-to-end (E2E) tests, use `docker-compose.e2e.yml`.
