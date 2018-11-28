FROM golang:1.10 AS builder
LABEL maintainer="kierranm@gmail.com" \
      description="Forwards prometheus DeadMansSwitch alerts to CloudWatch" \
      version="0.0.2"

RUN useradd -u 10001 deadmanswatch

# Copy the code from the host and compile it
WORKDIR $GOPATH/src/github.com/KierranM/deadmanswatch
COPY ./vendor $GOPATH/src/github.com/KierranM/deadmanswatch/vendor
COPY ./main.go $GOPATH/src/github.com/KierranM/deadmanswatch/main.go
COPY ./cmd $GOPATH/src/github.com/KierranM/deadmanswatch/cmd
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /deadmanswatch .

FROM alpine:latest AS cacerts
RUN apk add --update ca-certificates

FROM scratch
COPY --from=builder /deadmanswatch ./
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=cacerts /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER deadmanswatch
WORKDIR /
ENTRYPOINT ["./deadmanswatch"]
