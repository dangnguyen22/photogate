FROM golang:1.17-alpine

RUN apk update && apk add --no-cache sudo ngrep net-tools gcc make git libc-dev binutils-gold g++

ENV TZ=Asia/Ho_Chi_Minh

WORKDIR /src
ENV GOPATH=/src/.gopath \
	GOCACHE=$GOPATH/cache \
	GOPRIVATE=gitlab.sendo.vn \
	PATH=$PATH:$GOPATH/bin

USER root

ENV HISTFILE=$GOPATH/.bash_history
