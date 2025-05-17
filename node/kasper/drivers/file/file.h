#pragma once

#include <string>
#include "ifile.h"
#include "datapack.h"

class File : public IFile
{
public:
    std::string storageRoot;
    File(std::string sr);
    void writeFileToPoint(std::string pointId, std::string fileId, char *data, size_t len) override;
    DataPack readFileFromPoint(std::string pointId, std::string fileId) override;
    void writeFileToGlobal(std::string filePath, char *data, size_t len) override;
    DataPack readFileFromGlobal(std::string filePath) override;
    bool pointFileExists(std::string pointId, std::string fileId) override;
    bool globalFileExists(std::string filePath) override;
};
