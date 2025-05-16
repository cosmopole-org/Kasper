#include <string>
#include <iostream>
#include <filesystem>
#include <fstream>
#include "file.h"

namespace fs = std::filesystem;

File::File(std::string sr)
{
    try
    {
        if (fs::create_directories(sr))
        {
            std::cout << "Storage files root path directories created.\n";
        }
        else
        {
            std::cout << "Storage files root path directories already exist.\n";
        }
    }
    catch (const fs::filesystem_error &e)
    {
        std::cerr << "Error: " << e.what() << '\n';
    }
}

void File::writeFileToPoint(std::string pointId, std::string fileId, char *data, size_t len) {
    this->writeFileToGlobal(pointId + "/" + fileId, data, len);
}

DataPack File::readFileFromPoint(std::string pointId, std::string fileId) {
    return this->readFileFromGlobal(pointId + "/" + fileId);
}

void File::writeFileToGlobal(std::string filePath, char *data, size_t len) {
    std::ofstream file(this->storageRoot + "/" + filePath, std::ios::binary);
    if (file) {
        file.write(data, len);
    }
}

DataPack File::readFileFromGlobal(std::string filePath) {
    std::ifstream file(this->storageRoot + "/" + filePath, std::ios::binary | std::ios::ate);
    if (!file) {
        std::cerr << "Failed to open file\n";
        char data[0];
        return DataPack{
            data,
            0
        };
    }
    std::streamsize size = file.tellg();
    file.seekg(0, std::ios::beg);
    char* buffer = new char[size];
    if (!file.read(buffer, size)) {
        std::cerr << "Failed to read file\n";
        delete[] buffer;
        char data[0];
        return DataPack{
            data,
            0
        };
    }
    size_t len = size;
    return DataPack{
        buffer,
        len
    };
}

bool File::pointFileExists(std::string pointId, std::string fileId) {
    return this->globalFileExists(pointId + "/" + fileId);
}

bool File::globalFileExists(std::string filePath) {
    return fs::exists(this->storageRoot + "/" + filePath);
}
