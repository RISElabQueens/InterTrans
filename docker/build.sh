#!/bin/bash
docker build -t codetransengine/cpp-clang:latest ./cpp-clang
docker build -t codetransengine/golang:latest ./golang
docker build -t codetransengine/java:latest ./java
docker build -t codetransengine/node:latest ./node
docker build -t codetransengine/rust:latest ./rust
docker build -t codetransengine/python3:latest ./python3
