FROM ubuntu:18.04

WORKDIR /go

COPY gourl /go/
ADD templates /go

EXPOSE 80

CMD ["./gourl"]
