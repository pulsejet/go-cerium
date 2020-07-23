FROM golang:latest as builder

RUN go get -u github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/pulsejet/go-cerium
COPY ./Gopkg.toml ./Gopkg.lock ./
RUN dep ensure --vendor-only

RUN apt update && apt install -y git ca-certificates && update-ca-certificates

COPY . ./
RUN CGO_ENABLED=0 go build .

# ENTRYPOINT ["./go-cerium"]

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/pulsejet/go-cerium/go-cerium /
COPY --from=builder /go/src/github.com/pulsejet/go-cerium/.env /
ENTRYPOINT [ "/go-cerium" ]
