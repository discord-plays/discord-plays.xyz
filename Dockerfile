FROM golang:1.17

WORKDIR /go/src/app
COPY . .
RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 8080
CMD ["website"]
