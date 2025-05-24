#pragma once

#include "ichain.h"
#include <string>
#include <vector>
#include <map>
#include <chrono>
#include <set>
#include <mutex>
#include <thread> // For std::thread in Node class declaration
#include <queue>  // For std::queue in can_see

// Forward declarations for Boost.Asio components if used, but since we're using
// raw sockets, these are not strictly necessary in the header.
// However, if we were to define socket-related members in Node, we'd need them.
// For raw sockets, we just need the int server_socket_fd and std::thread.

// --- Placeholder Cryptography Types ---
using Hash = std::string;
using PublicKey = std::string;
using PrivateKey = std::string;
using Signature = std::string;

// --- ChainTransaction Structure ---
struct ChainTransaction {
    std::string typ_id;
    std::string payload_id;
    long long timestamp;

    std::string to_string() const;

    friend std::ostream& operator<<(std::ostream& os, const ChainTransaction& tx);
};

// --- Event Structure ---
struct Event {
    Hash self_parent_hash;
    Hash other_parent_hash;
    std::vector<ChainTransaction> transactions;
    PublicKey creator_public_key;
    Signature signature;
    long long timestamp;
    Hash event_hash;
    int round_num;
    bool is_witness;

    Event(); // Default constructor for deserialization
    Event(Hash self_parent, Hash other_parent, std::vector<ChainTransaction> txs, PublicKey creator_pub_key, PrivateKey creator_priv_key);

    bool verify() const;

    friend std::ostream& operator<<(std::ostream& os, const Event& event);
};

// --- Global Helper Functions (Cryptography & Serialization) ---
Hash hash_data(const std::string& data);
std::pair<PublicKey, PrivateKey> generate_key_pair();
Signature sign_data(const std::string& data, const PrivateKey& private_key);
bool verify_signature(const std::string& data, const Signature& signature, const PublicKey& public_key);

std::string serialize_transaction(const ChainTransaction& tx);
std::vector<std::string> split_string(const std::string& s, char delimiter);
std::string serialize_event(const Event& event);
Event deserialize_event(const std::string& data);

// --- Node Class ---
class Node : public IChain {
public:
    PublicKey public_key;
    PrivateKey private_key;
    std::string node_id;
    int port;
    std::set<std::string> known_peer_addresses;
    std::set<PublicKey> network_members;

    std::map<Hash, Event> event_dag;
    Hash last_self_event_hash;
    int current_consensus_round;

    // Networking components for raw sockets
    int server_socket_fd;
    std::thread server_thread;
    mutable std::mutex dag_mutex;

    Node(std::string id, int p, const std::vector<std::string>& initial_peers);
    ~Node();

    void handle_incoming_connection(int client_socket_fd, const std::string& remote_endpoint_str);
    void start_server();
    void start_gossip();
    Event create_event(const std::vector<ChainTransaction>& new_transactions);
    bool add_event(const Event& event);

    // Consensus-related functions
    void calculate_event_properties(const Hash& event_hash);
    bool can_see(const Hash& descendant_hash, const Hash& ancestor_hash) const;
    bool strongly_see(const Hash& event_x_hash, const Hash& witness_w_hash) const;
    std::set<Hash> find_witnesses_for_round(int round) const;
    std::set<Hash> elect_famous_witnesses(int round) const;
    std::vector<ChainTransaction> determine_event_order(int round, const std::set<Hash>& famous_witnesses_for_round) const;
    void try_advance_consensus();
	void submitTrx(std::string typ, std::string payload) override;

    friend std::ostream& operator<<(std::ostream& os, const Node& node);
};
