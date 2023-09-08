FROM golang:1.20 as builder

WORKDIR /workspace
COPY go.* ./

RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/echo-server .
USER nonroot:nonroot

ENTRYPOINT ["/echo-server"]
