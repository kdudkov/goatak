FROM golang:alpine AS builder

WORKDIR /build
COPY . .
RUN go mod tidy && go build -o dist/ ./cmd/...

FROM alpine

WORKDIR /app
COPY --from=builder /build/dist/goatak_server /app/goatak_server
COPY ./data /app/data
COPY ./goatak_server.yml /app/
COPY ./users.yml /app/
CMD ["./goatak_server"]