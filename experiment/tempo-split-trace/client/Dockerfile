FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY client/ ./client/
RUN CGO_ENABLED=0 go build -o /out/client ./client

FROM alpine:latest
COPY --from=build /out/client /usr/local/bin/client
ENTRYPOINT ["/usr/local/bin/client"]
