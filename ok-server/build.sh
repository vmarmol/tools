#!/bin/bash

go build .
docker build -t vmarmol/ok-server .
