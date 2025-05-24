#include "tools.h"

Tools::Tools(ICore *core, std::map<std::string, std::string> env)
{
    this->file = new File(env["STORAGE_ROOT"]);
    this->storage = new Storage(env["BASE_DB_PATH"]);
    this->signaler = new Signaler(core);
    this->security = new Security(core, env["STORAGE_ROOT"], this->file, this->storage, this->signaler);
    this->network = new Tcp(core);
    this->federation = new Fed(core);

    const int chainPort = 8082;
    std::vector<std::string> peersForA{};
    std::vector<std::string> allNodes = {"172.77.5.1:" + std::to_string(chainPort), "172.77.5.2:" + std::to_string(chainPort), "172.77.5.3:" + std::to_string(chainPort)};
    int myIndex = 0;
    for (auto item : peersForA) {
        if (item == core->getIp() + ":" + std::to_string(chainPort)) {
            continue;
        }
        peersForA.push_back(item);
    }
    Node *nodeA = new Node(core->getIp(), chainPort, peersForA);
    std::set<PublicKey> all_public_keys;
    all_public_keys.insert(nodeA->public_key);
    nodeA->network_members = all_public_keys;
    std::cout << std::endl;
    nodeA->start_server();
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    std::cout << std::endl;
    this->chain = nodeA;
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

IFed *Tools::getFederation()
{
    return this->federation;
}

IChain *Tools::getChain()
{
    return this->chain;
}