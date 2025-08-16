/* ========================================================================
 * Cross-Platform Implementation for StormDB
 * ======================================================================== */

#include "platform.h"
#include <stdarg.h>

#ifdef PLATFORM_WINDOWS
    #include <shlwapi.h>
    #include <psapi.h>
    #pragma comment(lib, "shlwapi.lib")
    #pragma comment(lib, "psapi.lib")
#endif

/* ========================================================================
 * Platform Information Functions
 * ======================================================================== */

const char* platform_get_os_name(void) {
    return PLATFORM_NAME;
}

const char* platform_get_arch_name(void) {
    return PLATFORM_ARCH_NAME;
}

const char* platform_get_compiler_name(void) {
    return PLATFORM_COMPILER_NAME;
}

int platform_get_compiler_version(void) {
    return PLATFORM_COMPILER_VERSION;
}

bool platform_is_64bit(void) {
#ifdef PLATFORM_64BIT
    return true;
#else
    return false;
#endif
}

/* ========================================================================
 * System Information Functions
 * ======================================================================== */

int platform_get_cpu_count(void) {
#ifdef PLATFORM_WINDOWS
    SYSTEM_INFO sysinfo;
    GetSystemInfo(&sysinfo);
    return (int)sysinfo.dwNumberOfProcessors;
#else
    long nprocs = sysconf(_SC_NPROCESSORS_ONLN);
    return (nprocs > 0) ? (int)nprocs : 1;
#endif
}

uint64_t platform_get_memory_size(void) {
#ifdef PLATFORM_WINDOWS
    MEMORYSTATUSEX status;
    status.dwLength = sizeof(status);
    if (GlobalMemoryStatusEx(&status)) {
        return status.ullTotalPhys;
    }
    return 0;
#elif defined(PLATFORM_LINUX)
    long pages = sysconf(_SC_PHYS_PAGES);
    long page_size = sysconf(_SC_PAGE_SIZE);
    if (pages > 0 && page_size > 0) {
        return (uint64_t)pages * (uint64_t)page_size;
    }
    return 0;
#elif defined(PLATFORM_MACOS)
    int64_t mem_size;
    size_t size = sizeof(mem_size);
    if (sysctlbyname("hw.memsize", &mem_size, &size, NULL, 0) == 0) {
        return (uint64_t)mem_size;
    }
    return 0;
#else
    return 0;
#endif
}

const char* platform_get_hostname(void) {
    static char hostname[PLATFORM_MAX_HOSTNAME];
    
#ifdef PLATFORM_WINDOWS
    DWORD size = sizeof(hostname);
    if (GetComputerNameA(hostname, &size)) {
        return hostname;
    }
#else
    if (gethostname(hostname, sizeof(hostname)) == 0) {
        hostname[sizeof(hostname) - 1] = '\0';
        return hostname;
    }
#endif
    return "unknown";
}

platform_pid_t platform_get_pid(void) {
#ifdef PLATFORM_WINDOWS
    return GetCurrentProcessId();
#else
    return getpid();
#endif
}

platform_pid_t platform_get_parent_pid(void) {
#ifdef PLATFORM_WINDOWS
    // Windows implementation would require more complex code
    return 0;
#else
    return getppid();
#endif
}

/* ========================================================================
 * File System Functions
 * ======================================================================== */

bool platform_path_exists(const char* path) {
    if (!path) return false;
    
#ifdef PLATFORM_WINDOWS
    return PathFileExistsA(path) != FALSE;
#else
    return access(path, F_OK) == 0;
#endif
}

bool platform_is_directory(const char* path) {
    if (!path) return false;
    
#ifdef PLATFORM_WINDOWS
    DWORD attrs = GetFileAttributesA(path);
    return (attrs != INVALID_FILE_ATTRIBUTES) && (attrs & FILE_ATTRIBUTE_DIRECTORY);
#else
    struct stat st;
    return (stat(path, &st) == 0) && S_ISDIR(st.st_mode);
#endif
}

bool platform_create_directory(const char* path) {
    if (!path) return false;
    
#ifdef PLATFORM_WINDOWS
    return CreateDirectoryA(path, NULL) != 0 || GetLastError() == ERROR_ALREADY_EXISTS;
#else
    return mkdir(path, 0755) == 0 || errno == EEXIST;
#endif
}

bool platform_remove_directory(const char* path) {
    if (!path) return false;
    
#ifdef PLATFORM_WINDOWS
    return RemoveDirectoryA(path) != 0;
#else
    return rmdir(path) == 0;
#endif
}

char* platform_get_current_directory(void) {
#ifdef PLATFORM_WINDOWS
    char* buffer = malloc(PLATFORM_MAX_PATH);
    if (buffer && GetCurrentDirectoryA(PLATFORM_MAX_PATH, buffer) > 0) {
        return buffer;
    }
    free(buffer);
    return NULL;
#else
    return getcwd(NULL, 0);
#endif
}

char* platform_get_executable_path(void) {
    char* path = malloc(PLATFORM_MAX_PATH);
    if (!path) return NULL;
    
#ifdef PLATFORM_WINDOWS
    if (GetModuleFileNameA(NULL, path, PLATFORM_MAX_PATH) > 0) {
        return path;
    }
#elif defined(PLATFORM_LINUX)
    ssize_t len = readlink("/proc/self/exe", path, PLATFORM_MAX_PATH - 1);
    if (len > 0) {
        path[len] = '\0';
        return path;
    }
#elif defined(PLATFORM_MACOS)
    uint32_t size = PLATFORM_MAX_PATH;
    if (_NSGetExecutablePath(path, &size) == 0) {
        return path;
    }
#endif
    
    free(path);
    return NULL;
}

char* platform_join_path(const char* base, const char* component) {
    if (!base || !component) return NULL;
    
    size_t base_len = strlen(base);
    size_t comp_len = strlen(component);
    size_t total_len = base_len + comp_len + 2; // +1 for separator, +1 for null terminator
    
    char* result = malloc(total_len);
    if (!result) return NULL;
    
    strcpy(result, base);
    
    // Add separator if needed
    if (base_len > 0 && base[base_len - 1] != PLATFORM_PATH_SEPARATOR) {
        result[base_len] = PLATFORM_PATH_SEPARATOR;
        result[base_len + 1] = '\0';
    }
    
    strcat(result, component);
    return result;
}

char* platform_normalize_path(const char* path) {
    if (!path) return NULL;
    
    char* normalized = malloc(strlen(path) + 1);
    if (!normalized) return NULL;
    
    strcpy(normalized, path);
    
    // Convert path separators to platform-specific ones
    for (char* p = normalized; *p; p++) {
#ifdef PLATFORM_WINDOWS
        if (*p == '/') *p = '\\';
#else
        if (*p == '\\') *p = '/';
#endif
    }
    
    return normalized;
}

/* ========================================================================
 * Process Functions
 * ======================================================================== */

bool platform_is_process_running(platform_pid_t pid) {
#ifdef PLATFORM_WINDOWS
    HANDLE process = OpenProcess(PROCESS_QUERY_INFORMATION, FALSE, pid);
    if (process) {
        DWORD exit_code;
        bool running = GetExitCodeProcess(process, &exit_code) && exit_code == STILL_ACTIVE;
        CloseHandle(process);
        return running;
    }
    return false;
#else
    return kill(pid, 0) == 0;
#endif
}

bool platform_kill_process(platform_pid_t pid, int sig) {
#ifdef PLATFORM_WINDOWS
    HANDLE process = OpenProcess(PROCESS_TERMINATE, FALSE, pid);
    if (process) {
        bool result = TerminateProcess(process, sig) != 0;
        CloseHandle(process);
        return result;
    }
    return false;
#else
    return kill(pid, sig) == 0;
#endif
}

/* ========================================================================
 * Thread Functions
 * ======================================================================== */

#ifdef PLATFORM_WINDOWS
typedef struct {
    void* (*start_routine)(void*);
    void* arg;
} thread_start_data_t;

static DWORD WINAPI thread_start_wrapper(LPVOID lpParam) {
    thread_start_data_t* data = (thread_start_data_t*)lpParam;
    void* (*start_routine)(void*) = data->start_routine;
    void* arg = data->arg;
    free(data);
    
    void* result = start_routine(arg);
    return (DWORD)(uintptr_t)result;
}
#endif

int platform_thread_create(platform_thread_t* thread, void* (*start_routine)(void*), void* arg) {
#ifdef PLATFORM_WINDOWS
    thread_start_data_t* data = malloc(sizeof(thread_start_data_t));
    if (!data) return -1;
    
    data->start_routine = start_routine;
    data->arg = arg;
    
    *thread = CreateThread(NULL, 0, thread_start_wrapper, data, 0, NULL);
    return (*thread != NULL) ? 0 : -1;
#else
    return pthread_create(thread, NULL, start_routine, arg);
#endif
}

int platform_thread_join(platform_thread_t thread, void** retval) {
#ifdef PLATFORM_WINDOWS
    DWORD result = WaitForSingleObject(thread, INFINITE);
    if (result == WAIT_OBJECT_0) {
        if (retval) {
            DWORD exit_code;
            GetExitCodeThread(thread, &exit_code);
            *retval = (void*)(uintptr_t)exit_code;
        }
        CloseHandle(thread);
        return 0;
    }
    return -1;
#else
    return pthread_join(thread, retval);
#endif
}

void platform_thread_exit(void* retval) {
#ifdef PLATFORM_WINDOWS
    ExitThread((DWORD)(uintptr_t)retval);
#else
    pthread_exit(retval);
#endif
}

platform_thread_t platform_thread_self(void) {
#ifdef PLATFORM_WINDOWS
    return GetCurrentThread();
#else
    return pthread_self();
#endif
}

/* ========================================================================
 * Mutex Functions
 * ======================================================================== */

int platform_mutex_init(platform_mutex_t* mutex) {
#ifdef PLATFORM_WINDOWS
    *mutex = CreateMutex(NULL, FALSE, NULL);
    return (*mutex != NULL) ? 0 : -1;
#else
    return pthread_mutex_init(mutex, NULL);
#endif
}

int platform_mutex_destroy(platform_mutex_t* mutex) {
#ifdef PLATFORM_WINDOWS
    return CloseHandle(*mutex) ? 0 : -1;
#else
    return pthread_mutex_destroy(mutex);
#endif
}

int platform_mutex_lock(platform_mutex_t* mutex) {
#ifdef PLATFORM_WINDOWS
    DWORD result = WaitForSingleObject(*mutex, INFINITE);
    return (result == WAIT_OBJECT_0) ? 0 : -1;
#else
    return pthread_mutex_lock(mutex);
#endif
}

int platform_mutex_unlock(platform_mutex_t* mutex) {
#ifdef PLATFORM_WINDOWS
    return ReleaseMutex(*mutex) ? 0 : -1;
#else
    return pthread_mutex_unlock(mutex);
#endif
}

int platform_mutex_trylock(platform_mutex_t* mutex) {
#ifdef PLATFORM_WINDOWS
    DWORD result = WaitForSingleObject(*mutex, 0);
    return (result == WAIT_OBJECT_0) ? 0 : -1;
#else
    return pthread_mutex_trylock(mutex);
#endif
}

/* ========================================================================
 * Time Functions
 * ======================================================================== */

uint64_t platform_get_timestamp_ms(void) {
#ifdef PLATFORM_WINDOWS
    return GetTickCount64();
#else
    struct timespec ts;
    if (clock_gettime(CLOCK_MONOTONIC, &ts) == 0) {
        return (uint64_t)ts.tv_sec * 1000 + (uint64_t)ts.tv_nsec / 1000000;
    }
    return 0;
#endif
}

uint64_t platform_get_timestamp_us(void) {
#ifdef PLATFORM_WINDOWS
    LARGE_INTEGER frequency, counter;
    if (QueryPerformanceFrequency(&frequency) && QueryPerformanceCounter(&counter)) {
        return (uint64_t)(counter.QuadPart * 1000000 / frequency.QuadPart);
    }
    return platform_get_timestamp_ms() * 1000;
#else
    struct timespec ts;
    if (clock_gettime(CLOCK_MONOTONIC, &ts) == 0) {
        return (uint64_t)ts.tv_sec * 1000000 + (uint64_t)ts.tv_nsec / 1000;
    }
    return 0;
#endif
}

void platform_sleep_ms(uint32_t milliseconds) {
#ifdef PLATFORM_WINDOWS
    Sleep(milliseconds);
#else
    usleep(milliseconds * 1000);
#endif
}

void platform_sleep_us(uint32_t microseconds) {
#ifdef PLATFORM_WINDOWS
    // Windows Sleep is only millisecond precision
    Sleep((microseconds + 999) / 1000);
#else
    usleep(microseconds);
#endif
}

/* ========================================================================
 * Dynamic Library Functions
 * ======================================================================== */

void* platform_dlopen(const char* filename) {
#ifdef PLATFORM_WINDOWS
    return LoadLibraryA(filename);
#else
    return dlopen(filename, RTLD_LAZY);
#endif
}

void* platform_dlsym(void* handle, const char* symbol) {
#ifdef PLATFORM_WINDOWS
    return GetProcAddress((HMODULE)handle, symbol);
#else
    return dlsym(handle, symbol);
#endif
}

int platform_dlclose(void* handle) {
#ifdef PLATFORM_WINDOWS
    return FreeLibrary((HMODULE)handle) ? 0 : -1;
#else
    return dlclose(handle);
#endif
}

const char* platform_dlerror(void) {
#ifdef PLATFORM_WINDOWS
    static char error_buffer[256];
    DWORD error = GetLastError();
    FormatMessageA(FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
                   NULL, error, 0, error_buffer, sizeof(error_buffer), NULL);
    return error_buffer;
#else
    return dlerror();
#endif
}

/* ========================================================================
 * File Operations
 * ======================================================================== */

platform_file_t platform_file_open(const char* path, int flags, int mode) {
    if (!path) return PLATFORM_INVALID_FILE;
    
#ifdef PLATFORM_WINDOWS
    DWORD access = 0;
    DWORD creation = 0;
    DWORD attributes = FILE_ATTRIBUTE_NORMAL;
    
    // Convert Unix-style flags to Windows
    if (flags & O_RDONLY) access |= GENERIC_READ;
    if (flags & O_WRONLY) access |= GENERIC_WRITE;
    if (flags & O_RDWR) access |= GENERIC_READ | GENERIC_WRITE;
    
    if (flags & O_CREAT) {
        if (flags & O_EXCL) {
            creation = CREATE_NEW;
        } else if (flags & O_TRUNC) {
            creation = CREATE_ALWAYS;
        } else {
            creation = OPEN_ALWAYS;
        }
    } else {
        if (flags & O_TRUNC) {
            creation = TRUNCATE_EXISTING;
        } else {
            creation = OPEN_EXISTING;
        }
    }
    
    return CreateFileA(path, access, FILE_SHARE_READ | FILE_SHARE_WRITE, 
                       NULL, creation, attributes, NULL);
#else
    return open(path, flags, mode);
#endif
}

int platform_file_close(platform_file_t file) {
#ifdef PLATFORM_WINDOWS
    return CloseHandle(file) ? 0 : -1;
#else
    return close(file);
#endif
}

ssize_t platform_file_read(platform_file_t file, void* buffer, size_t size) {
#ifdef PLATFORM_WINDOWS
    DWORD bytes_read;
    if (ReadFile(file, buffer, (DWORD)size, &bytes_read, NULL)) {
        return (ssize_t)bytes_read;
    }
    return -1;
#else
    return read(file, buffer, size);
#endif
}

ssize_t platform_file_write(platform_file_t file, const void* buffer, size_t size) {
#ifdef PLATFORM_WINDOWS
    DWORD bytes_written;
    if (WriteFile(file, buffer, (DWORD)size, &bytes_written, NULL)) {
        return (ssize_t)bytes_written;
    }
    return -1;
#else
    return write(file, buffer, size);
#endif
}

int platform_file_lock(platform_file_t file, bool exclusive) {
#ifdef PLATFORM_WINDOWS
    OVERLAPPED overlapped = {0};
    DWORD flags = exclusive ? LOCKFILE_EXCLUSIVE_LOCK : 0;
    flags |= LOCKFILE_FAIL_IMMEDIATELY;
    return LockFileEx(file, flags, 0, MAXDWORD, MAXDWORD, &overlapped) ? 0 : -1;
#else
    int operation = exclusive ? LOCK_EX : LOCK_SH;
    operation |= LOCK_NB; // Non-blocking
    return flock(file, operation);
#endif
}

int platform_file_unlock(platform_file_t file) {
#ifdef PLATFORM_WINDOWS
    OVERLAPPED overlapped = {0};
    return UnlockFileEx(file, 0, MAXDWORD, MAXDWORD, &overlapped) ? 0 : -1;
#else
    return flock(file, LOCK_UN);
#endif
}

/* ========================================================================
 * Signal Handling Functions
 * ======================================================================== */

#ifdef PLATFORM_WINDOWS
static BOOL WINAPI console_handler(DWORD signal) {
    switch (signal) {
        case CTRL_C_EVENT:
        case CTRL_BREAK_EVENT:
        case CTRL_CLOSE_EVENT:
        case CTRL_SHUTDOWN_EVENT:
            // Call the registered handler
            raise(SIGINT);
            return TRUE;
        default:
            return FALSE;
    }
}
#endif

int platform_signal_set_handler(int sig, platform_signal_handler_t handler) {
#ifdef PLATFORM_WINDOWS
    if (sig == SIGINT || sig == SIGTERM) {
        // Set up console control handler for Windows
        if (!SetConsoleCtrlHandler(console_handler, TRUE)) {
            return -1;
        }
    }
    return (signal(sig, handler) != SIG_ERR) ? 0 : -1;
#else
    return (signal(sig, handler) != SIG_ERR) ? 0 : -1;
#endif
}

int platform_signal_ignore(int sig) {
    return platform_signal_set_handler(sig, SIG_IGN);
}

int platform_signal_block(int sig) {
#ifdef PLATFORM_WINDOWS
    // Windows doesn't have direct signal blocking
    return 0;
#else
    sigset_t set;
    sigemptyset(&set);
    sigaddset(&set, sig);
    return sigprocmask(SIG_BLOCK, &set, NULL);
#endif
}

int platform_signal_unblock(int sig) {
#ifdef PLATFORM_WINDOWS
    // Windows doesn't have direct signal unblocking
    return 0;
#else
    sigset_t set;
    sigemptyset(&set);
    sigaddset(&set, sig);
    return sigprocmask(SIG_UNBLOCK, &set, NULL);
#endif
}

/* ========================================================================
 * Socket Operations
 * ======================================================================== */

int platform_socket_init(void) {
#ifdef PLATFORM_WINDOWS
    WSADATA wsaData;
    return WSAStartup(MAKEWORD(2, 2), &wsaData);
#else
    return 0;
#endif
}

void platform_socket_cleanup(void) {
#ifdef PLATFORM_WINDOWS
    WSACleanup();
#endif
}

platform_socket_t platform_socket_create(int domain, int type, int protocol) {
#ifdef PLATFORM_WINDOWS
    return socket(domain, type, protocol);
#else
    return socket(domain, type, protocol);
#endif
}

int platform_socket_close(platform_socket_t socket) {
#ifdef PLATFORM_WINDOWS
    return closesocket(socket);
#else
    return close(socket);
#endif
}

int platform_socket_bind(platform_socket_t socket, const struct sockaddr* addr, platform_socklen_t addrlen) {
    return bind(socket, addr, addrlen);
}

int platform_socket_listen(platform_socket_t socket, int backlog) {
    return listen(socket, backlog);
}

platform_socket_t platform_socket_accept(platform_socket_t socket, struct sockaddr* addr, platform_socklen_t* addrlen) {
    return accept(socket, addr, addrlen);
}

int platform_socket_connect(platform_socket_t socket, const struct sockaddr* addr, platform_socklen_t addrlen) {
    return connect(socket, addr, addrlen);
}

void platform_print_info(void) {
    printf("Platform Information:\n");
    printf("  OS: %s\n", platform_get_os_name());
    printf("  Architecture: %s (%s)\n", platform_get_arch_name(), 
           platform_is_64bit() ? "64-bit" : "32-bit");
    printf("  Compiler: %s (version %d)\n", platform_get_compiler_name(), 
           platform_get_compiler_version());
    printf("  CPU Cores: %d\n", platform_get_cpu_count());
    printf("  Memory: %.2f GB\n", platform_get_memory_size() / (1024.0 * 1024.0 * 1024.0));
    printf("  Hostname: %s\n", platform_get_hostname());
    printf("  Process ID: %d\n", (int)platform_get_pid());
}

bool platform_init(void) {
#ifdef PLATFORM_WINDOWS
    // Initialize Winsock
    WSADATA wsaData;
    int result = WSAStartup(MAKEWORD(2, 2), &wsaData);
    if (result != 0) {
        return false;
    }
#endif
    return true;
}

void platform_cleanup(void) {
#ifdef PLATFORM_WINDOWS
    WSACleanup();
#endif
}
