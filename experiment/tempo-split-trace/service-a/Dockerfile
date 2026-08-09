FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY service-a/ ./service-a/
RUN CGO_ENABLED=0 go build -o /out/service-a ./service-a

FROM alpine:latest
COPY --from=build /out/service-a /usr/local/bin/service-a
ENTRYPOINT ["/usr/local/bin/service-a"]
