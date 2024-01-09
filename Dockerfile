FROM public.ecr.aws/docker/library/golang:1.21 AS builder
LABEL maintainer="cmartin@wiley.com" \
      description="Forwards prometheus DeadMansSwitch alerts to CloudWatch" \
      version="1.0.1"

RUN useradd -u 10001 deadmanswatch

# Copy the code from the host and compile it
WORKDIR $GOPATH/src/github.com/KierranM/deadmanswatch
COPY ./go.mod ./go.sum $GOPATH/src/github.com/KierranM/deadmanswatch/
COPY ./main.go $GOPATH/src/github.com/KierranM/deadmanswatch/main.go
COPY ./cmd $GOPATH/src/github.com/KierranM/deadmanswatch/cmd
RUN go get
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix nocgo -o /deadmanswatch .

FROM alpine:latest AS cacerts
RUN apk add --update ca-certificates

FROM scratch
COPY --from=builder /deadmanswatch ./
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=cacerts /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER deadmanswatch
WORKDIR /
ENTRYPOINT ["./deadmanswatch"]
