FROM ubuntu as APP

RUN mkdir /app 
WORKDIR /app

ADD sts-wire /usr/local/bin/sts-wire

ENTRYPOINT ["sts-wire"]