#!/bin/bash
docker build -t intertrans/cpp-clang:latest ./cpp-clang
docker build -t intertrans/golang:latest ./golang
docker build -t intertrans/java:latest ./java
docker build -t intertrans/node:latest ./node
docker build -t intertrans/rust:latest ./rust
docker build -t intertrans/python3:latest ./python3
