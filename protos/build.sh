#!/bin/bash
protoc --go_out=. --go-grpc_out=. protos.proto
python -m grpc_tools.protoc -I. --python_out=../client/intertrans --grpc_python_out=../client/intertrans --pyi_out=../client/intertrans protos.proto