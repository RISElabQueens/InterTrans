#!/bin/bash
singularity build --force ./img/python3.sif docker-daemon://codetransengine/python3:latest
singularity build --force ./img/golang.sif docker-daemon://codetransengine/golang:latest
singularity build --force ./img/cpp-clang.sif docker-daemon://codetransengine/cpp-clang:latest
singularity build --force ./img/java.sif docker-daemon://codetransengine/java:latest
singularity build --force ./img/rust.sif docker-daemon://codetransengine/rust:latest
singularity build --force ./img/node.sif docker-daemon://codetransengine/node:latest