FROM golang:1.15 AS build

ADD . /opt/app
WORKDIR /opt/app
RUN go build ./main.go

FROM ubuntu:20.04

MAINTAINER Dmitry Kostrikin

RUN apt-get -y update && apt-get install -y tzdata

ENV PGVER 12
RUN apt-get -y update && apt-get install -y postgresql-$PGVER

USER postgres

RUN /etc/init.d/postgresql start &&\
    psql --command "CREATE USER yourname WITH SUPERUSER PASSWORD 'yourpassword';" &&\
    createdb -O yourname postgre &&\
    /etc/init.d/postgresql stop


EXPOSE 5432

VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

USER root

WORKDIR /usr/src/app

COPY . .
COPY --from=build /opt/app/main .

EXPOSE 5000
ENV PGPASSWORD yourpassword
CMD service postgresql start &&  psql -h localhost -d postgre -U yourname -p 5432 -a -q -f ./schema.sql && ./main
