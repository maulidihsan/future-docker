FROM golang:1.12 as build-env

RUN mkdir /app
WORKDIR /app
COPY go.mod .
COPY go.sum .

RUN go mod download
COPY main.go .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /go/bin/rest

FROM scratch
COPY --from=build-env /go/bin/rest /go/bin/rest
ENTRYPOINT ["/go/bin/rest"]