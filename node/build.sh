
g++ -std=c++17 -g main.cpp \
    -O2 -Iinclude $(find kasper/ -path kasper/drivers/elpis -prune -o -name "*.cpp" -print) \
    -I. -pthread -lssl -lcrypto -lrocksdb -lpthread -lz -lsnappy -lzstd -llz4 -lbz2 -lwasmedge -static-libgcc -static-libstdc++ \
    -o genesis
