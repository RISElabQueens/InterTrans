FROM esolang/base

COPY script /bin/script

RUN apk add --update nodejs npm && \
    mkdir /modules && cd /modules && \
    npm install readline-sync iostream hashmap lodash big-integer stdin readline map node-int64 fraction.js math mathjs long prompt-sync vector prompt-sync && \
    rm -rf /var/cache/apk/* /tmp/* && \
    ln -s /bin/script /bin/node && chmod +x /bin/script

ENV NODE_PATH=$NODE_PATH:/modules/node_modules