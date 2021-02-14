FROM golang:1.14 as builder
ARG GO_MODULES_TOKEN
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

FROM golang:latest

RUN apt-get update -qq
RUN apt-get install -y --no-install-recommends apt-utils

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && \
    apt-get -y install gcc mono-mcs && \
    rm -rf /var/lib/apt/lists/*
RUN apt install tesseract-ocr
RUN apt install libtesseract-dev
RUN apt-get install -y -qq libtesseract-dev libleptonica-dev
ENV TESSDATA_PREFIX=/usr/share/tesseract-ocr/4.00/tessdata/
RUN apt-get install -y -qq \
  tesseract-ocr-eng \
  tesseract-ocr-deu \
  tesseract-ocr-jpn
COPY --from=builder /go/src/app/superdentist-backend /usr/bin/

EXPOSE 8090

ENTRYPOINT ["/usr/bin/superdentist-backend"]
USER appuser
