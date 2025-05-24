#include "chain.h" // Include our custom header

#include <iostream>
#include <random>
#include <algorithm>
#include <sstream>
#include <iomanip>
#include <functional> // For std::bind

// POSIX Socket includes (only needed in implementation file)
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h> // For close()
#include <cstring>  // For memset
#include <errno.h>  // For errno

// --- Placeholder Cryptography Functions Implementations ---
// (Same as before, these need to be replaced with a robust crypto library)

// Static counter for unique keys, defined once in the .cpp file
static int key_counter_impl = 0;

Hash hash_data(const std::string &data)
{
    size_t hash_val = std::hash<std::string>{}(data);
    std::ostringstream oss;
    oss << std::hex << std::setw(64) << std::setfill('0') << hash_val;
    return oss.str();
}

std::pair<PublicKey, PrivateKey> generate_key_pair()
{
    key_counter_impl++; // Use the static counter defined here
    PublicKey pub = "PUB_KEY_" + std::to_string(key_counter_impl);
    PrivateKey priv = "PRIV_KEY_" + std::to_string(key_counter_impl);
    return {pub, priv};
}

Signature sign_data(const std::string &data, const PrivateKey &private_key)
{
    return hash_data(data + private_key + "_signed");
}

bool verify_signature(const std::string &data, const Signature &signature, const PublicKey &public_key)
{
    return signature == hash_data(data + public_key.substr(0, public_key.find("_")) + "_signed");
}

std::string ChainTransaction::to_string() const
{
    return typ_id + "|" + payload_id + "|" + std::to_string(timestamp);
}

std::ostream &operator<<(std::ostream &os, const ChainTransaction &tx)
{
    os << "  Sender: " << tx.typ_id << ", Recipient: " << tx.payload_id
       << ", Timestamp: " << tx.timestamp;
    return os;
}

// --- Event Structure Implementations ---
Event::Event() : self_parent_hash(""), other_parent_hash(""), creator_public_key(""), signature(""), timestamp(0), event_hash(""), round_num(-1), is_witness(false) {}

Event::Event(Hash self_parent, Hash other_parent, std::vector<ChainTransaction> txs, PublicKey creator_pub_key, PrivateKey creator_priv_key)
    : self_parent_hash(std::move(self_parent)),
      other_parent_hash(std::move(other_parent)),
      transactions(std::move(txs)),
      creator_public_key(std::move(creator_pub_key)),
      round_num(-1),
      is_witness(false)
{

    timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
                    std::chrono::system_clock::now().time_since_epoch())
                    .count();

    std::string event_content_for_hash = self_parent_hash + other_parent_hash;
    for (const auto &tx : transactions)
    {
        event_content_for_hash += tx.to_string();
    }
    event_content_for_hash += creator_public_key;
    event_content_for_hash += std::to_string(timestamp);

    event_hash = hash_data(event_content_for_hash);
    signature = sign_data(event_content_for_hash, creator_priv_key);
}

bool Event::verify() const
{
    std::string event_content_for_hash = self_parent_hash + other_parent_hash;
    for (const auto &tx : transactions)
    {
        event_content_for_hash += tx.to_string();
    }
    event_content_for_hash += creator_public_key;
    event_content_for_hash += std::to_string(timestamp);

    if (event_hash != hash_data(event_content_for_hash))
    {
        std::cerr << "Error: Event hash mismatch for event " << event_hash << std::endl;
        return false;
    }

    if (!verify_signature(event_content_for_hash, signature, creator_public_key))
    {
        std::cerr << "Error: Signature verification failed for event " << event_hash << std::endl;
        return false;
    }
    return true;
}

std::ostream &operator<<(std::ostream &os, const Event &event)
{
    os << "--- Event ---" << std::endl;
    os << "  Hash: " << event.event_hash << std::endl;
    os << "  Self Parent: " << event.self_parent_hash << std::endl;
    os << "  Other Parent: " << event.other_parent_hash << std::endl;
    os << "  Creator: " << event.creator_public_key << std::endl;
    os << "  Timestamp: " << event.timestamp << std::endl;
    os << "  Signature: " << event.signature << std::endl;
    os << "  Round: " << event.round_num << (event.is_witness ? " (Witness)" : "") << std::endl;
    os << "  ChainTransactions (" << event.transactions.size() << "):" << std::endl;
    for (const auto &tx : event.transactions)
    {
        os << tx << std::endl;
    }
    return os;
}

// --- Serialization/Deserialization Functions Implementations ---
std::string serialize_transaction(const ChainTransaction &tx)
{
    return tx.typ_id + "|" + tx.payload_id + "|" + std::to_string(tx.timestamp);
}

std::vector<std::string> split_string(const std::string &s, char delimiter)
{
    std::vector<std::string> tokens;
    std::string token;
    std::istringstream tokenStream(s);
    while (std::getline(tokenStream, token, delimiter))
    {
        tokens.push_back(token);
    }
    return tokens;
}

std::string serialize_event(const Event &event)
{
    std::ostringstream oss;
    oss << event.self_parent_hash << "|"
        << event.other_parent_hash << "|"
        << event.transactions.size();
    for (const auto &tx : event.transactions)
    {
        oss << "|" << serialize_transaction(tx);
    }
    oss << "|" << event.creator_public_key
        << "|" << event.signature
        << "|" << event.timestamp
        << "|" << event.event_hash
        << "|" << event.round_num
        << "|" << (event.is_witness ? "1" : "0");
    return oss.str();
}

Event deserialize_event(const std::string &data)
{
    Event event;
    std::vector<std::string> parts = split_string(data, '|');

    if (parts.size() < 9)
    {
        std::cerr << "Error: Malformed event data for deserialization (too few parts): " << data << std::endl;
        return Event();
    }

    size_t current_part_idx = 0;
    event.self_parent_hash = parts[current_part_idx++];
    event.other_parent_hash = parts[current_part_idx++];
    size_t num_txs = std::stoul(parts[current_part_idx++]);

    for (size_t i = 0; i < num_txs; ++i)
    {
        if (current_part_idx + 3 >= parts.size())
        {
            std::cerr << "Error: Not enough parts for transaction during deserialization." << std::endl;
            return Event();
        }
        std::string sender = parts[current_part_idx++];
        std::string recipient = parts[current_part_idx++];
        long long timestamp = std::stoll(parts[current_part_idx++]);
        event.transactions.emplace_back(ChainTransaction{sender, recipient, timestamp});
        event.transactions.back().timestamp = timestamp;
    }

    if (current_part_idx + 4 >= parts.size())
    {
        std::cerr << "Error: Not enough parts for event metadata during deserialization." << std::endl;
        return Event();
    }
    event.creator_public_key = parts[current_part_idx++];
    event.signature = parts[current_part_idx++];
    event.timestamp = std::stoll(parts[current_part_idx++]);
    event.event_hash = parts[current_part_idx++];
    event.round_num = std::stoi(parts[current_part_idx++]);
    event.is_witness = (parts[current_part_idx++] == "1");

    return event;
}

// --- Node Class Implementations ---
Node::Node(std::string id, int p, const std::vector<std::string> &initial_peers)
    : node_id(std::move(id)),
      port(p),
      last_self_event_hash("0"),
      current_consensus_round(-1),
      server_socket_fd(-1)
{

    auto keys = generate_key_pair();
    public_key = keys.first;
    private_key = keys.second;
    std::cout << "Node '" << node_id << "' created with Public Key: " << public_key << ", listening on port " << port << std::endl;
    network_members.insert(public_key);

    for (const auto &peer : initial_peers)
    {
        if (peer != "127.0.0.1:" + std::to_string(port))
        {
            known_peer_addresses.insert(peer);
        }
    }
}

Node::~Node()
{
    if (server_socket_fd != -1)
    {
        close(server_socket_fd);
    }
    if (server_thread.joinable())
    {
        server_thread.join();
    }
}

void Node::handle_incoming_connection(int client_socket_fd, const std::string &remote_endpoint_str)
{
    std::cout << "Node '" << node_id << "': Handling incoming connection from " << remote_endpoint_str << std::endl;

    char buffer[4096];
    ssize_t bytes_received = recv(client_socket_fd, buffer, sizeof(buffer) - 1, 0);

    if (bytes_received > 0)
    {
        buffer[bytes_received] = '\0';
        std::string received_data(buffer);

        if (!received_data.empty() && received_data.back() == '\n')
        {
            received_data.pop_back();
        }

        std::cout << "Node '" << node_id << "': Received from " << remote_endpoint_str << ": " << received_data << std::endl;

        Event received_event = deserialize_event(received_data);
        if (received_event.event_hash != "")
        {
            bool added_new_event = false;
            {
                std::lock_guard<std::mutex> lock(dag_mutex);
                added_new_event = add_event(received_event);
                if (added_new_event)
                {
                    calculate_event_properties(received_event.event_hash);
                }
            }

            if (added_new_event)
            {
                std::cout << "Node '" << node_id << "': New event received, triggering instant broadcast and consensus check." << std::endl;
                std::thread([this]()
                            { this->start_gossip(); })
                    .detach();
                std::thread([this]()
                            { this->try_advance_consensus(); })
                    .detach();
            }
        }
        else
        {
            std::cerr << "Node '" << node_id << "': Failed to deserialize received data: " << received_data << std::endl;
        }
    }
    else if (bytes_received == 0)
    {
        std::cout << "Node '" << node_id << "': Peer " << remote_endpoint_str << " disconnected." << std::endl;
    }
    else
    {
        perror(("Node '" + node_id + "': Receive error from " + remote_endpoint_str).c_str());
    }

    close(client_socket_fd);
    std::cout << "Node '" << node_id << "': Connection with " << remote_endpoint_str << " closed." << std::endl;
}

void Node::start_server()
{
    server_socket_fd = socket(AF_INET, SOCK_STREAM, 0);
    if (server_socket_fd == -1)
    {
        perror(("Node '" + node_id + "': Failed to create socket").c_str());
        return;
    }

    int opt = 1;
    if (setsockopt(server_socket_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt)) < 0)
    {
        perror(("Node '" + node_id + "': setsockopt SO_REUSEADDR failed").c_str());
        close(server_socket_fd);
        server_socket_fd = -1;
        return;
    }

    sockaddr_in server_addr;
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = INADDR_ANY;
    server_addr.sin_port = htons(port);

    if (bind(server_socket_fd, (struct sockaddr *)&server_addr, sizeof(server_addr)) == -1)
    {
        perror(("Node '" + node_id + "': Failed to bind socket").c_str());
        close(server_socket_fd);
        server_socket_fd = -1;
        return;
    }

    if (listen(server_socket_fd, 5) == -1)
    {
        perror(("Node '" + node_id + "': Failed to listen on socket").c_str());
        close(server_socket_fd);
        server_socket_fd = -1;
        return;
    }

    server_thread = std::thread([this]()
                                {
        std::cout << "Node '" << node_id << "': Server thread started, listening on port " << port << std::endl;
        while (true) {
            sockaddr_in client_addr;
            socklen_t client_addr_len = sizeof(client_addr);
            int client_socket_fd = accept(server_socket_fd, (struct sockaddr*)&client_addr, &client_addr_len);

            if (client_socket_fd == -1) {
                if (errno == EINVAL || errno == EBADF) {
                    std::cout << "Node '" << node_id << "': Server socket closed, stopping accept loop." << std::endl;
                    break;
                }
                perror(("Node '" + node_id + "': Accept error").c_str());
                continue;
            }

            char client_ip[INET_ADDRSTRLEN];
            inet_ntop(AF_INET, &(client_addr.sin_addr), client_ip, INET_ADDRSTRLEN);
            std::string remote_endpoint_str = std::string(client_ip) + ":" + std::to_string(ntohs(client_addr.sin_port));

            std::thread(&Node::handle_incoming_connection, this, client_socket_fd, remote_endpoint_str).detach();
        }
        std::cout << "Node '" << node_id << "': Server thread stopped." << std::endl; });
}

void Node::start_gossip()
{
    std::string event_data_to_send;
    {
        std::lock_guard<std::mutex> lock(dag_mutex);
        if (last_self_event_hash != "0" && event_dag.count(last_self_event_hash))
        {
            event_data_to_send = serialize_event(event_dag.at(last_self_event_hash)) + "\n";
        }
        else
        {
            std::cout << "Node '" << node_id << "': No self event to broadcast yet." << std::endl;
            return;
        }
    }

    if (known_peer_addresses.empty())
    {
        std::cout << "Node '" << node_id << "': No peers to gossip with." << std::endl;
        return;
    }

    std::cout << "Node '" << node_id << "': Initiating full broadcast gossip." << std::endl;

    for (const auto &peer_address_str : known_peer_addresses)
    {
        size_t colon_pos = peer_address_str.find(':');
        if (colon_pos == std::string::npos)
        {
            std::cerr << "Node '" << node_id << "': Invalid peer address format: " << peer_address_str << std::endl;
            continue;
        }
        std::string ip_str = peer_address_str.substr(0, colon_pos);
        int peer_port = std::stoi(peer_address_str.substr(colon_pos + 1));

        std::thread([this, ip_str, peer_port, peer_address_str, event_data_to_send]()
                    {
            int sock_fd = socket(AF_INET, SOCK_STREAM, 0);
            if (sock_fd == -1) {
                perror(("Node '" + node_id + "': Failed to create client socket for " + peer_address_str).c_str());
                return;
            }

            sockaddr_in peer_addr;
            memset(&peer_addr, 0, sizeof(peer_addr));
            peer_addr.sin_family = AF_INET;
            peer_addr.sin_port = htons(peer_port);
            if (inet_pton(AF_INET, ip_str.c_str(), &peer_addr.sin_addr) <= 0) {
                std::cerr << "Node '" << node_id << "': Invalid address/Address not supported for " << peer_address_str << std::endl;
                close(sock_fd);
                return;
            }

            if (connect(sock_fd, (struct sockaddr*)&peer_addr, sizeof(peer_addr)) == -1) {
                close(sock_fd);
                return;
            }

            std::cout << "Node '" << node_id << "': Connected to " << peer_address_str << ". Sending event." << std::endl;
            if (send(sock_fd, event_data_to_send.c_str(), event_data_to_send.length(), 0) == -1) {
                perror(("Node '" + node_id + "': Send error to " + peer_address_str).c_str());
            } else {
                std::cout << "Node '" << node_id << "': Successfully broadcast event to " << peer_address_str << std::endl;
            }

            close(sock_fd); })
            .detach();
    }
}

Event Node::create_event(const std::vector<ChainTransaction> &new_transactions)
{
    std::lock_guard<std::mutex> lock(dag_mutex);

    Hash chosen_other_parent_hash = "0";
    long long latest_other_event_timestamp = -1;

    for (const auto &pair : event_dag)
    {
        const Event &existing_event = pair.second;
        if (existing_event.creator_public_key != public_key)
        {
            if (existing_event.timestamp > latest_other_event_timestamp)
            {
                latest_other_event_timestamp = existing_event.timestamp;
                chosen_other_parent_hash = existing_event.event_hash;
            }
        }
    }

    Event new_event(last_self_event_hash, chosen_other_parent_hash, new_transactions, public_key, private_key);

    event_dag[new_event.event_hash] = new_event;
    last_self_event_hash = new_event.event_hash;

    std::cout << "Node '" << node_id << "' created new event: " << new_event.event_hash << std::endl;

    calculate_event_properties(new_event.event_hash);

    std::thread([this]()
                { this->start_gossip(); })
        .detach();
    std::thread([this]()
                { this->try_advance_consensus(); })
        .detach();

    return new_event;
}

bool Node::add_event(const Event &event)
{
    if (event_dag.count(event.event_hash))
    {
        return false;
    }
    if (!event.verify())
    {
        std::cerr << "Node '" << node_id << "': Received invalid event " << event.event_hash << std::endl;
        return false;
    }
    event_dag[event.event_hash] = event;
    std::cout << "Node '" << node_id << "': Added event " << event.event_hash << " to DAG." << std::endl;
    return true;
}

void Node::calculate_event_properties(const Hash &event_hash)
{
    if (!event_dag.count(event_hash))
    {
        std::cerr << "Error: Event " << event_hash << " not found in DAG to calculate properties." << std::endl;
        return;
    }

    Event &event = event_dag.at(event_hash);

    if (event.round_num != -1)
    {
        return;
    }

    int self_parent_round = -1;
    if (event.self_parent_hash != "0" && event_dag.count(event.self_parent_hash))
    {
        calculate_event_properties(event.self_parent_hash);
        self_parent_round = event_dag.at(event.self_parent_hash).round_num;
    }
    else if (event.self_parent_hash == "0")
    {
        self_parent_round = -1;
    }

    int other_parent_round = -1;
    if (event.other_parent_hash != "0" && event_dag.count(event.other_parent_hash))
    {
        calculate_event_properties(event.other_parent_hash);
        other_parent_round = event_dag.at(event.other_parent_hash).round_num;
    }
    else if (event.other_parent_hash == "0")
    {
        other_parent_round = -1;
    }

    event.round_num = std::max({0, self_parent_round, other_parent_round});

    if (event.self_parent_hash == "0" || (event_dag.count(event.self_parent_hash) && event_dag.at(event.self_parent_hash).round_num < event.round_num))
    {
        event.is_witness = true;
        std::cout << "Node '" << node_id << "': Event " << event.event_hash << " is a witness for round " << event.round_num << std::endl;
    }
    else
    {
        event.is_witness = false;
    }
}

bool Node::can_see(const Hash &descendant_hash, const Hash &ancestor_hash) const
{
    if (descendant_hash == ancestor_hash)
    {
        return true;
    }
    std::lock_guard<std::mutex> lock(dag_mutex);
    if (!event_dag.count(descendant_hash) || !event_dag.count(ancestor_hash))
    {
        return false;
    }

    std::queue<Hash> q;
    std::set<Hash> visited;

    q.push(descendant_hash);
    visited.insert(descendant_hash);

    while (!q.empty())
    {
        Hash current_hash = q.front();
        q.pop();

        const Event &current_event = event_dag.at(current_hash);

        if (current_event.self_parent_hash != "0")
        {
            if (current_event.self_parent_hash == ancestor_hash)
                return true;
            if (visited.find(current_event.self_parent_hash) == visited.end())
            {
                q.push(current_event.self_parent_hash);
                visited.insert(current_event.self_parent_hash);
            }
        }
        if (current_event.other_parent_hash != "0")
        {
            if (current_event.other_parent_hash == ancestor_hash)
                return true;
            if (visited.find(current_event.other_parent_hash) == visited.end())
            {
                q.push(current_event.other_parent_hash);
                visited.insert(current_event.other_parent_hash);
            }
        }
    }
    return false;
}

bool Node::strongly_see(const Hash &event_x_hash, const Hash &witness_w_hash) const
{
    std::lock_guard<std::mutex> lock(dag_mutex);
    if (!event_dag.count(event_x_hash) || !event_dag.count(witness_w_hash))
    {
        return false;
    }

    if (!can_see(event_x_hash, witness_w_hash))
    {
        return false;
    }

    const Event &witness_w = event_dag.at(witness_w_hash);
    int target_round = witness_w.round_num;

    std::map<PublicKey, bool> members_seen_through;
    int members_count = network_members.size();
    int supermajority_threshold = (2 * members_count / 3) + 1;

    for (const auto &pair : event_dag)
    {
        const Event &current_event = pair.second;
        if (current_event.round_num == target_round && current_event.creator_public_key != witness_w.creator_public_key)
        {
            if (can_see(event_x_hash, current_event.event_hash) && can_see(current_event.event_hash, witness_w_hash))
            {
                members_seen_through[current_event.creator_public_key] = true;
            }
        }
    }

    return members_seen_through.size() >= supermajority_threshold;
}

std::set<Hash> Node::find_witnesses_for_round(int round) const
{
    std::lock_guard<std::mutex> lock(dag_mutex);
    std::set<Hash> witnesses;
    std::map<PublicKey, bool> creator_has_witness_in_round;

    for (const auto &pair : event_dag)
    {
        const Event &event = pair.second;
        if (event.round_num == round && event.is_witness)
        {
            if (creator_has_witness_in_round.find(event.creator_public_key) == creator_has_witness_in_round.end())
            {
                witnesses.insert(event.event_hash);
                creator_has_witness_in_round[event.creator_public_key] = true;
            }
        }
    }
    return witnesses;
}

std::set<Hash> Node::elect_famous_witnesses(int round) const
{
    std::set<Hash> famous_witnesses;
    std::set<Hash> witnesses_in_round = find_witnesses_for_round(round);

    if (witnesses_in_round.empty())
    {
        return famous_witnesses;
    }

    std::cout << "Node '" << node_id << "': (Conceptual) Electing famous witnesses for round " << round << std::endl;

    if (last_self_event_hash != "0" && event_dag.count(last_self_event_hash))
    {
        for (const Hash &witness_hash : witnesses_in_round)
        {
            if (strongly_see(last_self_event_hash, witness_hash))
            {
                famous_witnesses.insert(witness_hash);
                std::cout << "  Witness " << witness_hash << " (creator: " << event_dag.at(witness_hash).creator_public_key << ") is conceptually famous." << std::endl;
            }
        }
    }

    return famous_witnesses;
}

std::vector<ChainTransaction> Node::determine_event_order(int round, const std::set<Hash> &famous_witnesses_for_round) const
{
    std::lock_guard<std::mutex> lock(dag_mutex);
    std::vector<ChainTransaction> ordered_transactions;

    std::cout << "Node '" << node_id << "': (Conceptual) Determining event order for round " << round << std::endl;

    std::vector<const Event *> events_to_order;
    for (const auto &pair : event_dag)
    {
        if (pair.second.round_num == round)
        {
            events_to_order.push_back(&pair.second);
        }
    }

    std::sort(events_to_order.begin(), events_to_order.end(), [](const Event *a, const Event *b)
              { return a->timestamp < b->timestamp; });

    for (const Event *event_ptr : events_to_order)
    {
        for (const auto &tx : event_ptr->transactions)
        {
            ordered_transactions.push_back(tx);
        }
    }

    std::sort(ordered_transactions.begin(), ordered_transactions.end(), [](const ChainTransaction &a, const ChainTransaction &b)
              { return a.timestamp < b.timestamp; });

    std::cout << "Node '" << node_id << "': Total " << ordered_transactions.size() << " transactions ordered for round " << round << std::endl;
    return ordered_transactions;
}

void Node::try_advance_consensus()
{
    std::cout << "Node '" << node_id << "': Attempting to advance consensus from round " << current_consensus_round << std::endl;
    int next_round_to_check = current_consensus_round + 1;

    while (true)
    {
        std::set<Hash> witnesses = find_witnesses_for_round(next_round_to_check);
        if (witnesses.empty())
        {
            std::cout << "Node '" << node_id << "': No witnesses found for round " << next_round_to_check << ". Cannot advance consensus." << std::endl;
            break;
        }

        std::set<Hash> famous_witnesses = elect_famous_witnesses(next_round_to_check);

        if (!famous_witnesses.empty())
        {
            std::cout << "Node '" << node_id << "': Famous witnesses elected for round " << next_round_to_check << ". Proceeding to order." << std::endl;
            std::vector<ChainTransaction> ordered_txs = determine_event_order(next_round_to_check, famous_witnesses);

            std::cout << "Node '" << node_id << "': Consensus reached for round " << next_round_to_check << ". "
                      << ordered_txs.size() << " transactions ordered." << std::endl;

            current_consensus_round = next_round_to_check;
            next_round_to_check++;
        }
        else
        {
            std::cout << "Node '" << node_id << "': No famous witnesses elected for round " << next_round_to_check << ". Cannot advance consensus." << std::endl;
            break;
        }
    }
}

std::ostream &operator<<(std::ostream &os, const Node &node)
{
    os << "Node ID: " << node.node_id << ", Public Key: " << node.public_key << ", Port: " << node.port << std::endl;
    os << "  Known Events: " << node.event_dag.size() << std::endl;
    os << "  Last Self Event: " << node.last_self_event_hash << std::endl;
    os << "  Current Consensus Round: " << node.current_consensus_round << std::endl;
    os << "  Known Peers (" << node.known_peer_addresses.size() << "): ";
    for (const auto &peer : node.known_peer_addresses)
    {
        os << peer << " ";
    }
    os << std::endl;
    os << "  Network Members (" << node.network_members.size() << "): ";
    for (const auto &member_key : node.network_members)
    {
        os << member_key << " ";
    }
    os << std::endl;
    return os;
}

// --- Main Function for Demonstration ---
void Node::submitTrx(std::string typ, std::string payload)
{
    std::vector<ChainTransaction> new_txs;
    new_txs.emplace_back(ChainTransaction{typ, payload, std::chrono::duration_cast<std::chrono::milliseconds>(std::chrono::system_clock::now().time_since_epoch()).count()});
    this->create_event(new_txs);
}
