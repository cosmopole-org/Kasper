#pragma once

#include <string>
#include <cstdint>
#include <vector>
#include <iostream>
#include <cstring>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <netinet/in.h>
#include <openssl/rsa.h>
#include <openssl/pem.h>
#include <openssl/bio.h>
#include <openssl/err.h>
#include <openssl/evp.h>
#include <mutex>

class Utils
{
public:
    std::mutex siglock;

    static Utils &getInstance();
    bool stringStartsWith(const std::string &s1, const std::string &s2);
    int parseDataAsInt(char* buffer);
    char * convertIntToData(int n);
    std::string getKasperNodeIPAddress();
    RSA *load_public_key_from_string(const char *keyStr);
    RSA *load_private_key_from_string(const char *keyStr);
    bool generateRsaKeyPair(std::string destDir);
    bool verify_signature_rsa(RSA *rsa,
                              char *data,
                              char *sign);
};
