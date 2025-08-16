#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>

int main(void) {
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) { perror("socket"); return 2; }
    struct sockaddr_in addr;
    addr.sin_family = AF_INET;
    addr.sin_port = htons(8080);
    addr.sin_addr.s_addr = inet_addr("127.0.0.1");
    if (connect(sock, (struct sockaddr*)&addr, sizeof(addr)) != 0) { perror("connect"); close(sock); return 3; }
    const char* req = "GET /health HTTP/1.1\r\nHost: localhost\r\n\r\n";
    send(sock, req, strlen(req), 0);
    char buf[4096];
    int r = recv(sock, buf, sizeof(buf)-1, 0);
    if (r <= 0) { perror("recv"); close(sock); return 4; }
    buf[r] = '\0';
    if (strstr(buf, "application/json") == NULL) { fprintf(stderr, "expected json\n"); close(sock); return 5; }
    if (strstr(buf, "db") == NULL) { fprintf(stderr, "expected db field\n"); close(sock); return 6; }
    printf("unit_api_health PASS\n");
    close(sock);
    return 0;
}
