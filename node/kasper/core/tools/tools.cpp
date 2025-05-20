#include "tools.h"

Tools::Tools(ICore *core, std::map<std::string, std::string> env)
{
    this->file = new File(env["STORAGE_ROOT"]);
    this->storage = new Storage(env["BASE_DB_PATH"]);
    this->signaler = new Signaler(core);
    this->security = new Security(core, env["STORAGE_ROOT"], this->file, this->storage, this->signaler);
    this->network = new Tcp(core);
}

IStorage *Tools::getStorage()
{
    return this->storage;
}

ISecurity *Tools::getSecurity()
{
    return this->security;
}

ISignaler *Tools::getSignaler()
{
    return this->signaler;
}

IFile *Tools::getFile()
{
    return this->file;
}

ITcp *Tools::getNetwork()
{
    return this->network;
}

IWasm *Tools::getWasm()
{
    return this->wasm;
}