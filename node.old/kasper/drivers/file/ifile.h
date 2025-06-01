#pragma once

#include <string>
#include "datapack.h"

class IFile
{
public:
    virtual ~IFile() = default;
    virtual void writeFileToPoint(std::string pointId, std::string fileId, char *data, size_t len) = 0;
    virtual DataPack readFileFromPoint(std::string pointId, std::string fileId) = 0;
    virtual void writeFileToGlobal(std::string filePath, char *data, size_t len) = 0;
    virtual DataPack readFileFromGlobal(std::string filePath) = 0;
    virtual bool pointFileExists(std::string pointId, std::string fileId) = 0;
    virtual bool globalFileExists(std::string filePath) = 0;
};
