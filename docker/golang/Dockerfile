FROM esolang/base

COPY script /bin/script

RUN apk add --update go musl-dev && \
    rm -rf /var/cache/apk/* /tmp/* && \
    ln -s /bin/script /bin/golang && chmod +x /bin/script


COPY script /bin/script
RUN mkdir /test && chmod +x /bin/script
COPY vendor /test/vendor
COPY go.mod /test/go.mod
COPY go.sum /test/go.sum