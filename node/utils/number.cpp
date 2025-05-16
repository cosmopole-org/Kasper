#include <cstdint>
#include <vector>

int parseDataAsInt(char buffer[4])
{
    return uint32_t((unsigned char)(buffer[0]) << 24 |
                    (unsigned char)(buffer[1]) << 16 |
                    (unsigned char)(buffer[2]) << 8 |
                    (unsigned char)(buffer[3]));
}

std::vector<char> convertIntToData(int n)
{
    std::vector<char> bytes = std::vector<char>(4, 0);
    bytes[0] = (n >> 24) & 0xFF;
    bytes[1] = (n >> 16) & 0xFF;
    bytes[2] = (n >> 8) & 0xFF;
    bytes[3] = n & 0xFF;
    return bytes;
}