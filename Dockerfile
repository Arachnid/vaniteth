FROM golang:1.7.4

RUN mkdir -p /go/src/app
WORKDIR /go/src/app
RUN go get github.com/arachnid/vaniteth && go install github.com/arachnid/vaniteth
ENTRYPOINT ["vaniteth"]
