# Cerium
`go-cerium` is the golang backend for the dangerously accurate Google Forms clone designed for IIT Bombay, [cerium](https://github.com/pulsejet/cerium).

## Development
Install dependencies using `dep ensure` and run the backend with `go run main.go`. You need to have `mongodb` running and environment variables set correctly in `.env`. You also need to generate the IITB SSO authentication token and set it in `.env`.

## Build
Use `go build` to generate an optimized build.

## License
Licensed under the Mozilla Public License 2.0
