FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY service-b/ ./service-b/
RUN CGO_ENABLED=0 go build -o /out/service-b ./service-b

FROM alpine:latest
COPY --from=build /out/service-b /usr/local/bin/service-b
ENTRYPOINT ["/usr/local/bin/service-b"]
