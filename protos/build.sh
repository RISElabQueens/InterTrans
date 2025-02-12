#!/bin/bash
protoc --go_out=. --go-grpc_out=. protos.proto
python -m grpc_tools.protoc -I. --python_out=../client/codetransengine --grpc_python_out=../client/codetransengine --pyi_out=../client/codetransengine protos.proto