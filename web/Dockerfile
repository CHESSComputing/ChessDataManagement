FROM cern/cc7-base:20181210
MAINTAINER Valentin Kuznetsov vkuznet@gmail.com

ENV WDIR=/data
ENV USER=_web

ADD install.sh $WDIR/install.sh

RUN yum update -y && yum clean all

# create bashs link to bash
RUN ln -s /bin/bash /usr/bin/bashs

# cd to workdir
WORKDIR $WDIR

# download golang and install it
RUN curl -k -L -O https://dl.google.com/go/go1.12.1.linux-amd64.tar.gz
RUN tar xfz go1.12.1.linux-amd64.tar.gz
RUN rm go1.12.1.linux-amd64.tar.gz
ENV GOROOT=$WDIR/go
ENV PATH="${GOROOT}/bin:${WDIR}:${PATH}"

# get go dependencies
ENV GOPATH=$WDIR/gopath
RUN mkdir -p $GOPATH
ENV PATH="${GOROOT}/bin:${WDIR}:${PATH}"
RUN go get github.com/sirupsen/logrus
RUN go get github.com/mattn/go-sqlite3
RUN go get github.com/go-sql-driver/mysql
RUN go get -d github.com/shirou/gopsutil/...
RUN go get -d gopkg.in/jcmturner/gokrb5.v7/...
RUN go get gopkg.in/mgo.v2/
RUN go get gopkg.in/mgo.v2/bson
RUN go get github.com/vkuznet/ChessDataManagement

# build exporters
WORKDIR $GOPATH/src/github.com/vkuznet/ChessDataManagement/web
RUN make

# setup final environment
ENV PATH="${WDIR}/bin:${GOPATH}/bin:${PATH}"

USER $USER
WORKDIR $WDIR
CMD ["./web"]
