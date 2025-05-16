#include <iostream>
#include <cstring>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <netinet/in.h>

std::string getMachineIPAddress() {
    const char* kGoogleDnsIp = "8.8.8.8";
    const int kDnsPort = 53;

    int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) {
        perror("socket");
        return "";
    }

    sockaddr_in serv{};
    serv.sin_family = AF_INET;
    serv.sin_port = htons(kDnsPort);
    inet_pton(AF_INET, kGoogleDnsIp, &serv.sin_addr);

    int err = connect(sock, (const sockaddr*)&serv, sizeof(serv));
    if (err < 0) {
        perror("connect");
        close(sock);
        return "";
    }

    sockaddr_in name{};
    socklen_t namelen = sizeof(name);
    err = getsockname(sock, (sockaddr*)&name, &namelen);
    if (err < 0) {
        perror("getsockname");
        close(sock);
        return "";
    }

    char buffer[INET_ADDRSTRLEN];
    const char* p = inet_ntop(AF_INET, &name.sin_addr, buffer, sizeof(buffer));
    close(sock);

    if (p != nullptr)
        return std::string(buffer);
    else
        return "";
}
