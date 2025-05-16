#include <openssl/rsa.h>
#include <openssl/pem.h>
#include <openssl/bio.h>
#include <openssl/err.h>
#include <openssl/evp.h>
#include <iostream>
#include <cstring>

RSA *load_public_key_from_string(const char *keyStr)
{
    BIO *bio = BIO_new_mem_buf(keyStr, -1); // -1 means null-terminated
    if (!bio)
    {
        std::cerr << "BIO_new_mem_buf failed\n";
        return nullptr;
    }

    RSA *rsa = PEM_read_bio_RSA_PUBKEY(bio, nullptr, nullptr, nullptr);
    BIO_free(bio);

    if (!rsa)
    {
        std::cerr << "PEM_read_bio_RSA_PUBKEY failed\n";
    }
    return rsa;
}

RSA *load_private_key_from_string(const char *keyStr)
{
    BIO *bio = BIO_new_mem_buf(keyStr, -1);
    if (!bio)
    {
        std::cerr << "BIO_new_mem_buf failed\n";
        return nullptr;
    }

    RSA *rsa = PEM_read_bio_RSAPrivateKey(bio, nullptr, nullptr, nullptr);
    BIO_free(bio);

    if (!rsa)
    {
        std::cerr << "PEM_read_bio_RSAPrivateKey failed\n";
    }
    return rsa;
}

bool generateRsaKeyPair(std::string destDir)
{
    std::string priPath = destDir + "/private.pem";
    std::string pubPath = destDir + "/public.pem";

    int bits = 2048;
    unsigned long e = RSA_F4;

    RSA *rsa = RSA_new();
    BIGNUM *bne = BN_new();
    if (!BN_set_word(bne, e))
    {
        std::cerr << "BN_set_word failed\n";
        return false;
    }

    if (!RSA_generate_key_ex(rsa, bits, bne, nullptr))
    {
        std::cerr << "RSA_generate_key_ex failed\n";
        return false;
    }

    // Save private key
    FILE *privFile = fopen(priPath.c_str(), "wb");
    if (!PEM_write_RSAPrivateKey(privFile, rsa, nullptr, nullptr, 0, nullptr, nullptr))
    {
        std::cerr << "Failed to write private key\n";
    }
    fclose(privFile);

    // Save public key
    FILE *pubFile = fopen(pubPath.c_str(), "wb");
    if (!PEM_write_RSA_PUBKEY(pubFile, rsa))
    {
        std::cerr << "Failed to write public key\n";
    }
    fclose(pubFile);

    // Clean up
    RSA_free(rsa);
    BN_free(bne);

    return true;
}

bool verify_signature_rsa(RSA *rsa,
                          char *data,
                          char *sign)
{
    const unsigned char *message = (const unsigned char *)data;
    size_t message_len = strlen((const char *)message);
    const unsigned char *signature = (const unsigned char *)sign;
    size_t signature_len = strlen((const char *)signature);

    unsigned char hash[SHA256_DIGEST_LENGTH];
    SHA256(message, message_len, hash);

    int result = RSA_verify(NID_sha256, hash, SHA256_DIGEST_LENGTH, signature, signature_len, rsa);
    if (result == 1)
    {
        return true;
    }
    else
    {
        std::cerr << "RSA_verify failed: " << ERR_error_string(ERR_get_error(), nullptr) << "\n";
        return false;
    }
}