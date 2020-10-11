FROM alpine:latest

RUN apk add --no-cache bash && \
    apk add --update tzdata && \
    apk add --no-cache ca-certificates && \
    addgroup -S appgroup && adduser -u 1000 -S appuser -G appgroup

COPY ./superdentist-backend /usr/bin/

EXPOSE 8090

ENTRYPOINT ["/usr/bin/superdentist-backend"]
USER appuser
