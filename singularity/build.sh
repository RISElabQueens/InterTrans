#!/bin/bash
singularity build --force ./img/python3.sif docker-daemon://intertrans/python3:latest
singularity build --force ./img/golang.sif docker-daemon://intertrans/golang:latest
singularity build --force ./img/cpp-clang.sif docker-daemon://intertrans/cpp-clang:latest
singularity build --force ./img/java.sif docker-daemon://intertrans/java:latest
singularity build --force ./img/rust.sif docker-daemon://intertrans/rust:latest
singularity build --force ./img/node.sif docker-daemon://intertrans/node:latest