FROM golang:1.14 as builder
ENV GO111MODULE=on
ENV GO_MODULES_TOKEN=$GO_MODULES_TOKEN
WORKDIR /go/src/app
RUN git config --global url."https://${GO_MODULES_TOKEN}:x-oauth-basic@github.com/superdentist/sdclients".insteadOf "https://github.com/superdentist/sdclients"

COPY go.mod .
COPY go.sum .
# Get dependencies - will also be cached if we won't change mod/sum
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o superdentist-backend .

FROM alpine:latest

RUN apk add --no-cache bash && \
    apk add --update tzdata && \
    apk add --no-cache ca-certificates && \
    addgroup -S appgroup && adduser -u 1000 -S appuser -G appgroup

COPY --from=builder /go/src/app/superdentist-backend /usr/bin/

EXPOSE 8090

ENTRYPOINT ["/usr/bin/superdentist-backend"]
USER appuser
