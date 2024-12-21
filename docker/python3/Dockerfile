FROM esolang/base

COPY script /bin/script

RUN apk add --update python3 && \
    rm -rf /var/cache/apk/* /tmp/* && \
    ln -s /bin/script /bin/python3 && chmod +x /bin/script


