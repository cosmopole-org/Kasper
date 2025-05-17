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

	Security(ICore *core, std::string storageRoot, IFile* file, IStorage *storage, ISignaler *signaler)
	{
		this->core = core;
		this->storageRoot = storageRoot;
		this->file = file;
		this->storage = storage;
		this->signaler = signaler;
		this->keys = {};
		this->loadKeys();
	}

	void loadKeys() override
	{
		try
		{
			fs::create_directories(this->storageRoot + "/" + keysFolderName);
			for (const auto &entry : fs::directory_iterator(this->storageRoot + "/" + keysFolderName))
			{
				if (entry.is_directory())
				{
					std::string tag = entry.path().filename().string();
					auto priKey = file->readFileFromGlobal(keysFolderName + "/" + tag + "/private.pem");
					auto pubKey = file->readFileFromGlobal(keysFolderName + "/" + tag + "/public.pem");
					this->keys[tag] = new KeyPack{priKey, pubKey};
				}
			}
		}
		catch (const fs::filesystem_error &e)
		{
			std::cerr << "Error: " << e.what() << '\n';
		}
		if (this->keys.find("server_key") == this->keys.end())
		{
			this->generateSecureKeyPair("server_key");
		}
	}

	void generateSecureKeyPair(std::string tag) override
	{
		fs::create_directories(this->storageRoot + "/" + keysFolderName + "/" + tag);
		Utils::getInstance().generateRsaKeyPair(this->storageRoot + "/" + keysFolderName + "/" + tag);
		auto priKey = file->readFileFromGlobal(keysFolderName + "/" + tag + "/private.pem");
		auto pubKey = file->readFileFromGlobal(keysFolderName + "/" + tag + "/public.pem");
		this->keys[tag] = new KeyPack{priKey, pubKey};
	}

	std::vector<DataPack> fetchKeyPair(std::string tag) override
	{
		return {this->keys[tag]->priKey, this->keys[tag]->pubKey};
	}

	SignVerifyRes authWithSignature(std::string userId, std::string packet, std::string signatureBase64) override
	{
		RSA *publicKey;
		this->core->modifyState([userId, &publicKey](StateTrx *trx)
								{ publicKey = trx->getPubKey(userId); });
		if (publicKey == NULL)
		{
			return {false, "", false};
		}
		if (!Utils::getInstance().verify_signature_rsa(publicKey, packet, signatureBase64))
		{
			return {false, "", false};
		}
		else
		{
			std::string userType = "";
			bool isGod = false;
			this->core->modifyState([&userType, &isGod, userId](StateTrx *trx)
									{
				auto typ = trx->getColumn("User", userId, "type");
				userType = std::string(typ.data, typ.len);
				isGod = (trx->getLink("god::" + userId) == "true"); });
			return {true, userType, isGod};
		}
	}

	bool hasAccessToPoint(std::string userId, std::string pointId) override
	{
		if (pointId == "")
		{
			return false;
		}
		bool found = false;
		this->core->modifyState([userId, pointId, &found](StateTrx *trx)
								{
			if (trx->getLink("memberof::"+userId+"::"+pointId) == "true") {
				found = true;
			} });
		return found;
	}
};
