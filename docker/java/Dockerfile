FROM esolang/base

COPY script /bin/script

ENV BUILD_PACKAGES="openjdk17-jdk unzip wget" \
    RUNTIME_PACKAGES=""

RUN apk add $BUILD_PACKAGES $RUNTIME_PACKAGES --repository http://dl-cdn.alpinelinux.org/alpine/edge/community
RUN    rm -rf /var/cache/apk/* /tmp/* && \
    ln -s /bin/script /bin/java && chmod +x /bin/script

RUN wget https://download2.gluonhq.com/openjfx/22.0.1/openjfx-22.0.1_linux-x64_bin-sdk.zip && \
unzip openjfx-22.0.1_linux-x64_bin-sdk.zip && mv ./javafx-sdk-22.0.1 /usr/lib/

RUN rm -rf openjfx-22.0.1_linux-x64_bin-sdk.zip 

RUN ls /usr/lib/javafx-sdk-22.0.1/lib

# add path of javac and java to 
ENV PATH="${PATH}:/usr/lib/jvm/default-jvm/bin"
