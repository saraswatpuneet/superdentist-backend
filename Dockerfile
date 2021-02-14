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
LABEL maintainer="Puneet Saraswat <puneet.saraswat10074@gmail.com>"

RUN apt-get update -qq

# You need librariy files and headers of tesseract and leptonica.
# When you miss these or LD_LIBRARY_PATH is not set to them,
# you would face an error: "tesseract/baseapi.h: No such file or directory"
RUN apt-get install -y -qq libtesseract-dev libleptonica-dev

# In case you face TESSDATA_PREFIX error, you minght need to set env vars
# to specify the directory where "tessdata" is located.
ENV TESSDATA_PREFIX=/usr/share/tesseract-ocr/4.00/tessdata/

# Load languages.
# These {lang}.traineddata would b located under ${TESSDATA_PREFIX}/tessdata.
RUN apt-get install -y -qq \
  tesseract-ocr-eng \
  tesseract-ocr-deu \
  tesseract-ocr-jpn
# See https://github.com/tesseract-ocr/tessdata for the list of available languages.
# If you want to download these traineddata via `wget`, don't forget to locate
# downloaded traineddata under ${TESSDATA_PREFIX}/tessdata.

RUN go get -t github.com/otiai10/gosseract
RUN cd ${GOPATH}/src/github.com/otiai10/gosseract && go test

COPY --from=builder /go/src/app/superdentist-backend /usr/bin/
EXPOSE 8090

ENTRYPOINT ["/usr/bin/superdentist-backend"]
USER appuser
