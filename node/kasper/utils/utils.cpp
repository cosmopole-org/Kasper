
#include "utils.h"

Utils &Utils::getInstance()
{
    static Utils instance;
    return instance;
}

RSA *Utils::load_public_key_from_string(const char *keyStr)
{
    BIO *bio = BIO_new_mem_buf(keyStr, -1);
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

RSA *Utils::load_private_key_from_string(const char *keyStr)
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

bool Utils::generateRsaKeyPair(std::string destDir)
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

bool Utils::verify_signature_rsa(RSA *rsa,
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

std::string Utils::getKasperNodeIPAddress()
{
    const char *kGoogleDnsIp = "8.8.8.8";
    const int kDnsPort = 53;

    int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0)
    {
        perror("socket");
        return "";
    }

    sockaddr_in serv{};
    serv.sin_family = AF_INET;
    serv.sin_port = htons(kDnsPort);
    inet_pton(AF_INET, kGoogleDnsIp, &serv.sin_addr);

    int err = connect(sock, (const sockaddr *)&serv, sizeof(serv));
    if (err < 0)
    {
        perror("connect");
        close(sock);
        return "";
    }

    sockaddr_in name{};
    socklen_t namelen = sizeof(name);
    err = getsockname(sock, (sockaddr *)&name, &namelen);
    if (err < 0)
    {
        perror("getsockname");
        close(sock);
        return "";
    }

    char buffer[INET_ADDRSTRLEN];
    const char *p = inet_ntop(AF_INET, &name.sin_addr, buffer, sizeof(buffer));
    close(sock);

    if (p != nullptr)
        return std::string(buffer);
    else
        return "";
}

bool Utils::stringStartsWith(const std::string &s1, const std::string &s2)
{
    return s1.compare(0, s2.length(), s2) == 0;
}

int Utils::parseDataAsInt(char* buffer)
{
    return uint32_t((unsigned char)(buffer[0]) << 24 |
                    (unsigned char)(buffer[1]) << 16 |
                    (unsigned char)(buffer[2]) << 8 |
                    (unsigned char)(buffer[3]));
}

char *Utils::convertIntToData(int n)
{
    char* bytes = new char[4];
    bytes[3] = n & 0xFF;
    bytes[2] = (n >> 8) & 0xFF;
    bytes[1] = (n >> 16) & 0xFF;
    bytes[0] = (n >> 24) & 0xFF;
    return bytes;
}
