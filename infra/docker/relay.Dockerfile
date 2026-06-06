# syntax=docker/dockerfile:1.7

FROM golang:1.26.1-bookworm AS build

WORKDIR /src

COPY go.mod go.sum go.work ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/iseelocal-relay ./cmd/iseelocal-relay

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /
COPY --from=build /out/iseelocal-relay /usr/local/bin/iseelocal-relay

USER nonroot:nonroot
EXPOSE 8080 8081

ENTRYPOINT ["/usr/local/bin/iseelocal-relay"]
