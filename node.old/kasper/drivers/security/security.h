#pragma once

#include <string>
#include <iostream>
#include <filesystem>
#include <vector>
#include <map>
#include "../../core/core/icore.h"
#include "../file/ifile.h"
#include "../storage/istorage.h"
#include "../signaler/isignaler.h"
#include "../file/ifile.h"
#include "isecurity.h"
#include "../../utils/utils.h"

namespace fs = std::filesystem;

const std::string keysFolderName = "keys";

class Security : public ISecurity
{
public:
	ICore *core;
	IStorage *storage;
	ISignaler *signaler;
	IFile *file;
	std::string storageRoot;
	std::map<std::string, KeyPack *> keys;

	Security(ICore *core, std::string storageRoot, IFile* file, IStorage *storage, ISignaler *signaler);
	void loadKeys() override;
	std::vector<DataPack> generateSecureKeyPair(std::string tag) override;
	std::vector<DataPack> fetchKeyPair(std::string tag) override;
	SignVerifyRes authWithSignature(std::string userId, std::string packet, std::string signatureBase64) override;
	bool hasAccessToPoint(std::string userId, std::string pointId) override;
};
