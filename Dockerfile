FROM golang:1.24

WORKDIR /go/src/app
COPY . .
RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 8080
CMD ["discord-plays-xyz"]
