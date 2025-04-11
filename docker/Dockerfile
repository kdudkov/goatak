FROM golang:alpine AS builder

ARG branch
ARG commit

WORKDIR /build
COPY . .
RUN go mod tidy && go build -o dist/ -ldflags "-s -X main.gitRevision=$commit -X main.gitBranch=$branch" ./cmd/...

FROM alpine

EXPOSE 8080/tcp
EXPOSE 8088/tcp
EXPOSE 8089/tcp
EXPOSE 8443/tcp
EXPOSE 8446/tcp
EXPOSE 8999/tcp
EXPOSE 8999/udp

WORKDIR /app
RUN apk add --no-cache bash openssl zip
COPY --from=builder /build/dist/goatak_server /app/goatak_server
COPY ./cert/*.sh /app/cert/
COPY ./data /app/data
COPY ./docker/run.sh /app/
COPY ./goatak_server.yml /app/
CMD ["./run.sh"]