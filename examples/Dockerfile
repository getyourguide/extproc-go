FROM golang:1.25-bullseye AS build

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY examples/ examples/
COPY filter/ filter/
COPY service/ service/
COPY test/ test/
COPY server/ server/

RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -o extproc-go examples/main.go

RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -o healthcheck examples/hack/healthcheck.go

FROM gcr.io/distroless/base

COPY --from=build --chown=nonroot:nonroot /build/extproc-go .
COPY --from=build --chown=nonroot:nonroot /build/healthcheck .

CMD ["./extproc-go"]
