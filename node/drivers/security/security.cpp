#include <string>
#include <iostream>
#include <filesystem>
#include <vector>
#include <map>
#include "../../utils/crypto.cpp"
#include "../../core/core/icore.h"
#include "../storage/istorage.h"
#include "../signaler/isignaler.h"
#include "../file/ifile.h"
#include "isecurity.h"

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
	std::map<std::string, DataPack[2]> keys;
	
	Security(ICore *core, std::string storageRoot, IStorage *storage, ISignaler *signaler)
	{
		this->core = core;
		this->storageRoot = storageRoot;
		this->storage = storage;
		this->signaler = signaler;
		this->keys = {};
		this->loadKeys();
	}

	void loadKeys() override
	{
		try
		{
			for (const auto &entry : fs::directory_iterator(this->storageRoot + "/keys"))
			{
				if (entry.is_directory())
				{
					std::string tag = entry.path().filename().string();
					auto priKey = file->readFileFromGlobal("keys/" + tag + "/private.pem");
					auto pubKey = file->readFileFromGlobal("keys/" + tag + "/public.pem");
					this->keys.insert(tag, {});
					this->keys[tag][0] = priKey;
					this->keys[tag][1] = pubKey;
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
		generateRsaKeyPair(this->storageRoot + "/" + keysFolderName + "/" + tag);
		auto priKey = file->readFileFromGlobal("keys/" + tag + "/private.pem");
		auto pubKey = file->readFileFromGlobal("keys/" + tag + "/public.pem");
		this->keys.insert(tag, {});
		this->keys[tag][0] = priKey;
		this->keys[tag][1] = pubKey;
	}

	std::vector<DataPack> fetchKeyPair(std::string tag) override
	{
		return {this->keys[tag][0], this->keys[tag][1]};
	}

	SignVerifyRes authWithSignature(std::string userId, char *packet, char *signatureBase64) override
	{
		RSA *publicKey;
		this->core->modifyState([userId, &publicKey](StateTrx *trx)
								{ publicKey = trx->getPubKey(userId); });
		if (publicKey == NULL)
		{
			return {false, "", false};
		}
		if (!verify_signature_rsa(publicKey, packet, signatureBase64))
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
