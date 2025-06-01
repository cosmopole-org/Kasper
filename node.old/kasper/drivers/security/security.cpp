#include "security.h"

namespace fs = std::filesystem;

Security::Security(ICore *core, std::string storageRoot, IFile *file, IStorage *storage, ISignaler *signaler)
{
	this->core = core;
	this->storageRoot = storageRoot;
	this->file = file;
	this->storage = storage;
	this->signaler = signaler;
	this->keys = {};
	this->loadKeys();
}

void Security::loadKeys()
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

std::vector<DataPack> Security::generateSecureKeyPair(std::string tag)
{
	fs::create_directories(this->storageRoot + "/" + keysFolderName + "/" + tag);
	Utils::getInstance().generateRsaKeyPair(this->storageRoot + "/" + keysFolderName + "/" + tag);
	auto priKey = file->readFileFromGlobal(keysFolderName + "/" + tag + "/private.pem");
	auto pubKey = file->readFileFromGlobal(keysFolderName + "/" + tag + "/public.pem");
	this->keys[tag] = new KeyPack{priKey, pubKey};
	return {this->keys[tag]->priKey, this->keys[tag]->pubKey};
}

std::vector<DataPack> Security::fetchKeyPair(std::string tag)
{
	return {this->keys[tag]->priKey, this->keys[tag]->pubKey};
}

SignVerifyRes Security::authWithSignature(std::string userId, std::string packet, std::string signatureBase64)
{
	EVP_PKEY *publicKey;
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

bool Security::hasAccessToPoint(std::string userId, std::string pointId)
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
