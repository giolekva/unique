#!/bin/sh

make server_controller
make server_worker

./server_controller \
    --port=4321 \
    --num-bits=1024 \
    --start-from=https://gio.lekva.me \
    --num-documents=100 &


./server_worker \
    --controller-address=127.0.0.1:4321 \
    --num-workers=2 \
    --num-bits=1024 \
    --name=worker1 \
    --address=127.0.0.1 \
    --port=1234 &
