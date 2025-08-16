#ifndef PLATFORM_H
#define PLATFORM_H

/* ========================================================================
 * Cross-Platform Compatibility Layer for StormDB
 * Supports: Windows, macOS, Linux on x86_64, ARM64, ARM32
 * ======================================================================== */

#ifdef __cplusplus
extern "C" {
#endif

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <stdint.h>
#include <time.h>
#include <errno.h>

/* ========================================================================
 * Platform Detection
 * ======================================================================== */

/* Operating System Detection */
#if defined(_WIN32) || defined(_WIN64)
    #define PLATFORM_WINDOWS 1
    #define PLATFORM_NAME "Windows"
    #if defined(_WIN64)
        #define PLATFORM_64BIT 1
    #else
        #define PLATFORM_32BIT 1
    #endif
#elif defined(__APPLE__) && defined(__MACH__)
    #define PLATFORM_MACOS 1
    #define PLATFORM_UNIX 1
    #define PLATFORM_NAME "macOS"
    #include <TargetConditionals.h>
    #if TARGET_OS_IPHONE || TARGET_OS_SIMULATOR
        #define PLATFORM_IOS 1
    #endif
#elif defined(__linux__)
    #define PLATFORM_LINUX 1
    #define PLATFORM_UNIX 1
    #define PLATFORM_NAME "Linux"
#elif defined(__unix__) || defined(__unix)
    #define PLATFORM_UNIX 1
    #define PLATFORM_NAME "Unix"
#else
    #define PLATFORM_UNKNOWN 1
    #define PLATFORM_NAME "Unknown"
#endif

/* CPU Architecture Detection */
#if defined(__x86_64__) || defined(__x86_64) || defined(__amd64__) || defined(__amd64) || defined(_M_X64)
    #define PLATFORM_ARCH_X86_64 1
    #define PLATFORM_ARCH_NAME "x86_64"
    #define PLATFORM_64BIT 1
#elif defined(__i386__) || defined(__i386) || defined(_M_IX86)
    #define PLATFORM_ARCH_X86 1
    #define PLATFORM_ARCH_NAME "x86"
    #define PLATFORM_32BIT 1
#elif defined(__aarch64__) || defined(_M_ARM64)
    #define PLATFORM_ARCH_ARM64 1
    #define PLATFORM_ARCH_NAME "ARM64"
    #define PLATFORM_64BIT 1
#elif defined(__arm__) || defined(_M_ARM)
    #define PLATFORM_ARCH_ARM 1
    #define PLATFORM_ARCH_NAME "ARM"
    #define PLATFORM_32BIT 1
#else
    #define PLATFORM_ARCH_UNKNOWN 1
    #define PLATFORM_ARCH_NAME "Unknown"
#endif

/* Compiler Detection */
#if defined(_MSC_VER)
    #define PLATFORM_COMPILER_MSVC 1
    #define PLATFORM_COMPILER_NAME "MSVC"
    #define PLATFORM_COMPILER_VERSION _MSC_VER
#elif defined(__clang__)
    #define PLATFORM_COMPILER_CLANG 1
    #define PLATFORM_COMPILER_NAME "Clang"
    #define PLATFORM_COMPILER_VERSION (__clang_major__ * 10000 + __clang_minor__ * 100 + __clang_patchlevel__)
#elif defined(__GNUC__)
    #define PLATFORM_COMPILER_GCC 1
    #define PLATFORM_COMPILER_NAME "GCC"
    #define PLATFORM_COMPILER_VERSION (__GNUC__ * 10000 + __GNUC_MINOR__ * 100 + __GNUC_PATCHLEVEL__)
#else
    #define PLATFORM_COMPILER_UNKNOWN 1
    #define PLATFORM_COMPILER_NAME "Unknown"
    #define PLATFORM_COMPILER_VERSION 0
#endif

/* ========================================================================
 * Platform-Specific Includes
 * ======================================================================== */

#ifdef PLATFORM_WINDOWS
    #define WIN32_LEAN_AND_MEAN
    #include <windows.h>
    #include <winsock2.h>
    #include <ws2tcpip.h>
    #include <io.h>
    #include <direct.h>
    #include <process.h>
    #pragma comment(lib, "ws2_32.lib")
    #pragma comment(lib, "advapi32.lib")
#else
    #include <unistd.h>
    #include <pthread.h>
    #include <signal.h>
    #include <sys/types.h>
    #include <sys/stat.h>
    #include <sys/time.h>
    #include <fcntl.h>
    #include <dlfcn.h>
    #include <netinet/in.h>
    #include <sys/socket.h>
    #include <arpa/inet.h>
    #ifdef PLATFORM_LINUX
        #include <sys/file.h>
        #include <sys/prctl.h>
    #endif
    #ifdef PLATFORM_MACOS
        #include <libproc.h>
        #include <sys/param.h>
        #include <sys/sysctl.h>
    #endif
#endif

/* ========================================================================
 * Cross-Platform Type Definitions
 * ======================================================================== */

#ifdef PLATFORM_WINDOWS
    typedef HANDLE platform_thread_t;
    typedef HANDLE platform_mutex_t;
    typedef HANDLE platform_file_t;
    typedef DWORD platform_pid_t;
    typedef SOCKET platform_socket_t;
    typedef int platform_socklen_t;
    #define PLATFORM_INVALID_SOCKET INVALID_SOCKET
    #define PLATFORM_INVALID_FILE INVALID_HANDLE_VALUE
    #define PLATFORM_PATH_SEPARATOR '\\'
    #define PLATFORM_PATH_SEPARATOR_STR "\\"
    #define PLATFORM_LINE_ENDING "\r\n"
#else
    typedef pthread_t platform_thread_t;
    typedef pthread_mutex_t platform_mutex_t;
    typedef int platform_file_t;
    typedef pid_t platform_pid_t;
    typedef int platform_socket_t;
    typedef socklen_t platform_socklen_t;
    #define PLATFORM_INVALID_SOCKET -1
    #define PLATFORM_INVALID_FILE -1
    #define PLATFORM_PATH_SEPARATOR '/'
    #define PLATFORM_PATH_SEPARATOR_STR "/"
    #define PLATFORM_LINE_ENDING "\n"
#endif

/* ========================================================================
 * Cross-Platform Constants
 * ======================================================================== */

#define PLATFORM_MAX_PATH 4096
#define PLATFORM_MAX_HOSTNAME 256

/* Default paths */
#ifdef PLATFORM_WINDOWS
    #define PLATFORM_DEFAULT_CONFIG_DIR "C:\\ProgramData\\StormDB"
    #define PLATFORM_DEFAULT_LOG_DIR "C:\\ProgramData\\StormDB\\logs"
    #define PLATFORM_DEFAULT_PID_FILE "C:\\ProgramData\\StormDB\\stormdb.pid"
    #define PLATFORM_DEFAULT_PLUGIN_DIR "C:\\Program Files\\StormDB\\plugins"
    #define PLATFORM_LIBRARY_EXTENSION ".dll"
    #define PLATFORM_EXECUTABLE_EXTENSION ".exe"
#else
    #define PLATFORM_DEFAULT_CONFIG_DIR "/etc/stormdb"
    #define PLATFORM_DEFAULT_LOG_DIR "/var/log/stormdb"
    #define PLATFORM_DEFAULT_PID_FILE "/var/run/stormdb.pid"
    #define PLATFORM_DEFAULT_PLUGIN_DIR "/usr/lib/stormdb/plugins"
    #ifdef PLATFORM_MACOS
        #define PLATFORM_LIBRARY_EXTENSION ".dylib"
    #else
        #define PLATFORM_LIBRARY_EXTENSION ".so"
    #endif
    #define PLATFORM_EXECUTABLE_EXTENSION ""
#endif

/* ========================================================================
 * Function Declarations
 * ======================================================================== */

/* Platform information */
const char* platform_get_os_name(void);
const char* platform_get_arch_name(void);
const char* platform_get_compiler_name(void);
int platform_get_compiler_version(void);
bool platform_is_64bit(void);

/* System information */
int platform_get_cpu_count(void);
uint64_t platform_get_memory_size(void);
const char* platform_get_hostname(void);
platform_pid_t platform_get_pid(void);
platform_pid_t platform_get_parent_pid(void);

/* File system operations */
bool platform_path_exists(const char* path);
bool platform_is_directory(const char* path);
bool platform_create_directory(const char* path);
bool platform_remove_directory(const char* path);
char* platform_get_current_directory(void);
char* platform_get_executable_path(void);
char* platform_join_path(const char* base, const char* component);
char* platform_normalize_path(const char* path);

/* Process operations */
bool platform_is_process_running(platform_pid_t pid);
bool platform_kill_process(platform_pid_t pid, int sig);
int platform_spawn_process(const char* command, char* const argv[]);

/* Thread operations */
int platform_thread_create(platform_thread_t* thread, void* (*start_routine)(void*), void* arg);
int platform_thread_join(platform_thread_t thread, void** retval);
void platform_thread_exit(void* retval);
platform_thread_t platform_thread_self(void);

/* Mutex operations */
int platform_mutex_init(platform_mutex_t* mutex);
int platform_mutex_destroy(platform_mutex_t* mutex);
int platform_mutex_lock(platform_mutex_t* mutex);
int platform_mutex_unlock(platform_mutex_t* mutex);
int platform_mutex_trylock(platform_mutex_t* mutex);

/* Signal handling */
typedef void (*platform_signal_handler_t)(int);
// Signal handling
int platform_signal_set_handler(int sig, platform_signal_handler_t handler);
int platform_signal_ignore(int sig);
int platform_signal_block(int sig);
int platform_signal_unblock(int sig);

/* File operations */
platform_file_t platform_file_open(const char* path, int flags, int mode);
int platform_file_close(platform_file_t file);
ssize_t platform_file_read(platform_file_t file, void* buffer, size_t size);
ssize_t platform_file_write(platform_file_t file, const void* buffer, size_t size);
int platform_file_lock(platform_file_t file, bool exclusive);
int platform_file_unlock(platform_file_t file);

/* Socket operations */
int platform_socket_init(void);
void platform_socket_cleanup(void);
platform_socket_t platform_socket_create(int domain, int type, int protocol);
int platform_socket_close(platform_socket_t socket);
int platform_socket_bind(platform_socket_t socket, const struct sockaddr* addr, platform_socklen_t addrlen);
int platform_socket_listen(platform_socket_t socket, int backlog);
platform_socket_t platform_socket_accept(platform_socket_t socket, struct sockaddr* addr, platform_socklen_t* addrlen);
int platform_socket_connect(platform_socket_t socket, const struct sockaddr* addr, platform_socklen_t addrlen);

/* Time operations */
uint64_t platform_get_timestamp_ms(void);
uint64_t platform_get_timestamp_us(void);
void platform_sleep_ms(uint32_t milliseconds);
void platform_sleep_us(uint32_t microseconds);

/* Dynamic library operations */
void* platform_dlopen(const char* filename);
void* platform_dlsym(void* handle, const char* symbol);
int platform_dlclose(void* handle);
const char* platform_dlerror(void);

/* Utility functions */
void platform_print_info(void);
bool platform_init(void);
void platform_cleanup(void);

#ifdef __cplusplus
}
#endif

#endif /* PLATFORM_H */
