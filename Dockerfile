FROM alpine:3.8
MAINTAINER Philipp Hellmich <phil@hellmi.de>

RUN apk add --no-cache tini python3 python3-dev build-base && \
    mkdir /app

WORKDIR /app

COPY setup.py /app
COPY exporter.py /app
COPY lists.txt /app

RUN pip3 install .

USER nobody

ENTRYPOINT ["/sbin/tini", "--"]

CMD [ "python3", "exporter.py" ]
