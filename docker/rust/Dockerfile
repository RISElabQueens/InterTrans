# FROM esolang/build-base

# RUN apk add --update rust && \
#     rm -rf /var/cache/apk/* /tmp/* && \
#     ln -s /bin/script /bin/rust

# COPY script /root/script

FROM esolang/build-base

# Install rustup (Rust toolchain installer)
RUN apk add --no-cache \
    curl \
    gcc \
    musl-dev \
    openssl-dev \
    libgcc && \
    curl https://sh.rustup.rs -sSf | sh -s -- -y

# Ensure the rust toolchain is in the PATH
ENV PATH="/root/.cargo/bin:${PATH}"

COPY script /bin/script

# Cleanup
RUN rm -rf /var/cache/apk/* /tmp/* && chmod +x /bin/script && \
    ln -s /bin/script /bin/rust

RUN mkdir -p /root/src
COPY Cargo.toml /root/Cargo.toml

# Copy the cargo configuration file into the container
COPY config.toml /root/.cargo/config.toml

# Copy the vendor directory into the container
COPY vendor vendor


