
FROM golang:1.14.0-alpine3.11 as builder

WORKDIR /opt/app
RUN apk --update add curl git make gcc libc-dev

# Cached layer
COPY ./go.mod ./go.sum ./
RUN go mod download

# Sources dependent layer
COPY ./ ./
# badger needs cgo
#RUN CGO_ENABLED=0 
RUN GOOS=linux go build -ldflags "-X main.version=`git describe --tags --always`" -a ./cmd/sfs/

FROM alpine:3.11.2

ENV DOCKERFILE_VERSION  200216

WORKDIR /opt/app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /opt/app/sfs /usr/bin/sfs
COPY --from=builder /opt/app/html ./html

EXPOSE 8080
ENTRYPOINT ["/usr/bin/sfs"]
