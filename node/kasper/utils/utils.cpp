
#include "utils.h"

namespace fs = std::filesystem;

Utils &Utils::getInstance()
{
    static Utils instance;
    return instance;
}

EVP_PKEY *Utils::load_public_key_from_string(const std::string &key_str)
{
    BIO *bio = BIO_new_mem_buf(key_str.data(), static_cast<int>(key_str.size()));
    if (!bio)
    {
        std::cerr << "Failed to create BIO\n";
        return nullptr;
    }

    EVP_PKEY *pubkey = PEM_read_bio_PUBKEY(bio, nullptr, nullptr, nullptr);
    BIO_free(bio);

    if (!pubkey)
    {
        std::cerr << "Failed to parse public key from string\n";
        ERR_print_errors_fp(stderr);
    }

    return pubkey;
}

EVP_PKEY *Utils::load_private_key_from_string(const std::string &key_str)
{
    BIO *bio = BIO_new_mem_buf(key_str.data(), static_cast<int>(key_str.size()));
    if (!bio)
    {
        std::cerr << "Failed to create BIO\n";
        return nullptr;
    }

    EVP_PKEY *pkey = PEM_read_bio_PrivateKey(bio, nullptr, nullptr, nullptr);
    BIO_free(bio);

    if (!pkey)
    {
        std::cerr << "Failed to parse private key from string\n";
        ERR_print_errors_fp(stderr);
    }
    return pkey;
}

bool Utils::generateRsaKeyPair(std::string destDir)
{
    fs::create_directories(destDir);

    std::string priPath = destDir + "/private.pem";
    std::string pubPath = destDir + "/public.pem";

    EVP_PKEY *pkey = nullptr;
    EVP_PKEY_CTX *ctx = EVP_PKEY_CTX_new_id(EVP_PKEY_RSA, nullptr);
    if (!ctx)
    {
        std::cerr << "Failed to create context\n";
        ERR_print_errors_fp(stderr);
        return 1;
    }

    if (EVP_PKEY_keygen_init(ctx) <= 0)
    {
        std::cerr << "Failed to initialize keygen\n";
        ERR_print_errors_fp(stderr);
        EVP_PKEY_CTX_free(ctx);
        return 1;
    }

    if (EVP_PKEY_CTX_set_rsa_keygen_bits(ctx, 2048) <= 0)
    {
        std::cerr << "Failed to set RSA key size\n";
        ERR_print_errors_fp(stderr);
        EVP_PKEY_CTX_free(ctx);
        return 1;
    }

    if (EVP_PKEY_keygen(ctx, &pkey) <= 0)
    {
        std::cerr << "Failed to generate key\n";
        ERR_print_errors_fp(stderr);
        EVP_PKEY_CTX_free(ctx);
        return 1;
    }

    FILE *priv_fp = fopen(priPath.c_str(), "wb");
    PEM_write_PrivateKey(priv_fp, pkey, nullptr, nullptr, 0, nullptr, nullptr);
    fclose(priv_fp);

    FILE *pub_fp = fopen(pubPath.c_str(), "wb");
    PEM_write_PUBKEY(pub_fp, pkey);
    fclose(pub_fp);

    // Cleanup
    EVP_PKEY_free(pkey);
    EVP_PKEY_CTX_free(ctx);

    return true;
}

std::vector<unsigned char> base64_decode(const std::string &base64_input)
{
    BIO *bio = BIO_new_mem_buf(base64_input.data(), base64_input.size());
    BIO *b64 = BIO_new(BIO_f_base64());
    bio = BIO_push(b64, bio);

    // Don't expect newlines
    BIO_set_flags(bio, BIO_FLAGS_BASE64_NO_NL);

    std::vector<unsigned char> buffer(base64_input.size()); // max possible size
    int decoded_len = BIO_read(bio, buffer.data(), buffer.size());

    BIO_free_all(bio);

    if (decoded_len <= 0)
        return {};
    buffer.resize(decoded_len);
    return buffer;
}

bool Utils::verify_signature_rsa(EVP_PKEY *pubkey,
                                 std::string data,
                                 std::string sign)
{
    const unsigned char *message = reinterpret_cast<const unsigned char*>(data.data());
    size_t message_len = data.size();

    auto signatureBuffer = base64_decode(sign);
    const unsigned char *signature = signatureBuffer.data();
    size_t signature_len = signatureBuffer.size();

    EVP_MD_CTX *ctx = EVP_MD_CTX_new();
    if (!ctx)
        return false;

    bool result = false;

    // Init verify context with digest and public key
    if (EVP_DigestVerifyInit(ctx, nullptr, EVP_sha256(), nullptr, pubkey) <= 0)
    {

        EVP_MD_CTX_free(ctx);
        return result;
    }

    // Add message to be verified
    if (EVP_DigestVerifyUpdate(ctx, message, message_len) <= 0)
    {
        EVP_MD_CTX_free(ctx);
        return result;
    }

    // Final verification
    if (EVP_DigestVerifyFinal(ctx, signature, signature_len) == 1)
    {
        result = true; // Signature is valid
    }
    else
    {
        ERR_print_errors_fp(stderr); // Print OpenSSL error stack
    }

    EVP_MD_CTX_free(ctx);
    return result;
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

int Utils::parseDataAsInt(char *buffer)
{
    return uint32_t((unsigned char)(buffer[0]) << 24 |
                    (unsigned char)(buffer[1]) << 16 |
                    (unsigned char)(buffer[2]) << 8 |
                    (unsigned char)(buffer[3]));
}

char *Utils::convertIntToData(int n)
{
    char *bytes = new char[4];
    bytes[3] = n & 0xFF;
    bytes[2] = (n >> 8) & 0xFF;
    bytes[1] = (n >> 16) & 0xFF;
    bytes[0] = (n >> 24) & 0xFF;
    return bytes;
}
