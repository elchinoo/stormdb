#include "api.h"
#include "logging.h"
#include "platform.h"
#include "database.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

static volatile bool api_running = false;
static platform_thread_t api_thread;
static int api_port = 0;
static platform_socket_t listen_sock = PLATFORM_INVALID_SOCKET;

static void send_response(platform_socket_t s, const char* body, const char* content_type) {
    char hdr[512];
    int len = snprintf(hdr, sizeof(hdr), "HTTP/1.1 200 OK\r\nContent-Type: %s\r\nContent-Length: %zu\r\n\r\n", content_type, strlen(body));
    (void)send(s, hdr, len, 0);
    (void)send(s, body, (int)strlen(body), 0);
}

static void* api_thread_fn(void* arg) {
    (void)arg;
    LOG_INFO_MSG("API thread starting on port %d", api_port);
    if (platform_socket_init() != 0) {
        LOG_ERROR_MSG("Socket init failed");
        api_running = false;
        return NULL;
    }
    listen_sock = platform_socket_create(AF_INET, SOCK_STREAM, 0);
    if (listen_sock == PLATFORM_INVALID_SOCKET) {
        LOG_ERROR_MSG("Socket create failed");
        api_running = false;
        return NULL;
    }
    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = htonl(INADDR_ANY);
    addr.sin_port = htons((uint16_t)api_port);
    if (platform_socket_bind(listen_sock, (struct sockaddr*)&addr, sizeof(addr)) != 0) {
        LOG_ERROR_MSG("Bind failed on port %d", api_port);
        platform_socket_close(listen_sock);
        listen_sock = PLATFORM_INVALID_SOCKET;
        api_running = false;
        return NULL;
    }
    if (platform_socket_listen(listen_sock, 16) != 0) {
        LOG_ERROR_MSG("Listen failed");
        platform_socket_close(listen_sock);
        listen_sock = PLATFORM_INVALID_SOCKET;
        api_running = false;
        return NULL;
    }
    LOG_INFO_MSG("API listening on port %d", api_port);
    while (api_running) {
        struct sockaddr_in cli;
        platform_socklen_t clilen = sizeof(cli);
        platform_socket_t s = platform_socket_accept(listen_sock, (struct sockaddr*)&cli, &clilen);
        if (s == PLATFORM_INVALID_SOCKET) {
            platform_sleep_ms(50);
            continue;
        }
        // Read request (very small read, we only need the method+path)
        char buf[1024];
        int r = recv(s, buf, sizeof(buf)-1, 0);
        if (r <= 0) { platform_socket_close(s); continue; }
        buf[r] = '\0';
        // Look for GET /health
        if (strncmp(buf, "GET /health", 11) == 0) {
            unsigned long failures = database_get_reconnect_failures();
            unsigned long successes = database_get_reconnect_successes();
            const char* last_err = database_get_last_error();
            int connected = database_is_connected() ? 1 : 0;
            char body[1024];
            snprintf(body, sizeof(body), "{\"db\":{\"connected\":%d,\"reconnect_failures\":%lu,\"reconnect_successes\":%lu,\"last_error\":\"%s\"}}",
                     connected, failures, successes, last_err ? last_err : "");
            send_response(s, body, "application/json");
            platform_socket_close(s);
            continue;
        }
        // Fallback: simple hello
        const char* resp = "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 12\r\n\r\nHello Storm\n";
        (void)send(s, resp, (int)strlen(resp), 0);
        platform_socket_close(s);
    }
    if (listen_sock != PLATFORM_INVALID_SOCKET) {
        platform_socket_close(listen_sock);
        listen_sock = PLATFORM_INVALID_SOCKET;
    }
    platform_socket_cleanup();
    LOG_INFO_MSG("API thread exiting");
    return NULL;
}

bool api_start(int port) {
    if (api_running) return true;
    api_port = port;
    api_running = true;
    return platform_thread_create(&api_thread, api_thread_fn, NULL) == 0;
}

void api_stop(void) {
    if (!api_running) return;
    api_running = false;
    void* rv = NULL;
    platform_thread_join(api_thread, &rv);
}

bool api_restart(int new_port) {
    api_stop();
    return api_start(new_port);
}
