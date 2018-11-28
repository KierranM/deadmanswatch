FROM golang:1.10 AS builder
LABEL maintainer="kierranm@gmail.com" \
      description="Forwards prometheus DeadMansSwitch alerts to CloudWatch" \
      version="0.0.1"

RUN useradd -u 10001 deadmanswatch

# Copy the code from the host and compile it
WORKDIR $GOPATH/src/github.com/kierranm/deadmanswatch
COPY ./vendor $GOPATH/src/github.com/kierranm/deadmanswatch/vendor
COPY ./main.go $GOPATH/src/github.com/kierranm/deadmanswatch/main.go
COPY ./cmd $GOPATH/src/github.com/kierranm/deadmanswatch/cmd
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /deadmanswatch .

FROM scratch
COPY --from=builder /deadmanswatch ./
COPY --from=builder /etc/passwd /etc/passwd
USER deadmanswatch
WORKDIR /
ENTRYPOINT ["./deadmanswatch"]
