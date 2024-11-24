FROM golang:1.23-bullseye AS build

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY examples/ examples/
COPY filter/ filter/
COPY service/ service/
COPY httptest/ httptest/

RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -o extproc-go examples/main.go

FROM gcr.io/distroless/base

COPY --from=build --chown=nonroot:nonroot /build/extproc-go .

CMD ["./extproc-go"]
