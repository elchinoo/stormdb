#include "pidfile.h"
#include "logging.h"
#include "platform.h"

static char *current_pid_file = NULL;
static platform_file_t pid_fd = PLATFORM_INVALID_FILE;

bool pidfile_create(const char *pid_file) {
    if (!pid_file) {
        LOG_ERROR_MSG("PID file path is NULL");
        return false;
    }
    
    // Check if another instance is already running
    if (pidfile_check_running(pid_file)) {
        LOG_ERROR_MSG("Another instance of StormDB is already running");
        return false;
    }
    
    // Open PID file for writing
#ifdef PLATFORM_WINDOWS
    pid_fd = platform_file_open(pid_file, O_WRONLY | O_CREAT | O_TRUNC, 0644);
#else
    pid_fd = open(pid_file, O_WRONLY | O_CREAT | O_TRUNC, 0644);
#endif
    if (pid_fd == PLATFORM_INVALID_FILE) {
        LOG_ERROR_MSG("Failed to create PID file %s: %s", pid_file, strerror(errno));
        return false;
    }
    
    // Lock the file
    if (platform_file_lock(pid_fd, true) != 0) {
        LOG_ERROR_MSG("Failed to lock PID file %s: %s", pid_file, strerror(errno));
        platform_file_close(pid_fd);
        pid_fd = PLATFORM_INVALID_FILE;
        return false;
    }
    
    // Write current PID
    char pid_str[32];
    platform_pid_t current_pid = platform_get_pid();
    int len = snprintf(pid_str, sizeof(pid_str), "%d\n", (int)current_pid);
    if (platform_file_write(pid_fd, pid_str, len) != len) {
        LOG_ERROR_MSG("Failed to write PID to file %s: %s", pid_file, strerror(errno));
        platform_file_close(pid_fd);
        pid_fd = PLATFORM_INVALID_FILE;
#ifdef PLATFORM_WINDOWS
        DeleteFileA(pid_file);
#else
        unlink(pid_file);
#endif
        return false;
    }
    
    // Store path for cleanup
    current_pid_file = strdup(pid_file);
    
    LOG_INFO_MSG("Created PID file %s with PID %d", pid_file, (int)current_pid);
    return true;
}

void pidfile_remove(void) {
    if (pid_fd != PLATFORM_INVALID_FILE) {
        platform_file_unlock(pid_fd);
        platform_file_close(pid_fd);
        pid_fd = PLATFORM_INVALID_FILE;
    }
    
    if (current_pid_file) {
#ifdef PLATFORM_WINDOWS
        if (DeleteFileA(current_pid_file)) {
#else
        if (unlink(current_pid_file) == 0) {
#endif
            LOG_INFO_MSG("Removed PID file %s", current_pid_file);
        } else {
            LOG_WARN_MSG("Failed to remove PID file %s: %s", current_pid_file, strerror(errno));
        }
        free(current_pid_file);
        current_pid_file = NULL;
    }
}

bool pidfile_check_running(const char *pid_file) {
    if (!pid_file) {
        return false;
    }
    
#ifdef PLATFORM_WINDOWS
    platform_file_t fd = platform_file_open(pid_file, O_RDONLY, 0);
#else
    int fd = open(pid_file, O_RDONLY);
#endif
    if (fd == PLATFORM_INVALID_FILE) {
        // File doesn't exist, no instance running
        return false;
    }
    
    char pid_str[32];
#ifdef PLATFORM_WINDOWS
    ssize_t bytes_read = platform_file_read(fd, pid_str, sizeof(pid_str) - 1);
    platform_file_close(fd);
#else
    ssize_t bytes_read = read(fd, pid_str, sizeof(pid_str) - 1);
    close(fd);
#endif
    
    if (bytes_read <= 0) {
        // Invalid PID file, remove it
#ifdef PLATFORM_WINDOWS
        DeleteFileA(pid_file);
#else
        unlink(pid_file);
#endif
        return false;
    }
    
    pid_str[bytes_read] = '\0';
    platform_pid_t existing_pid = (platform_pid_t)atoi(pid_str);
    
    // Check if process is still running
    if (platform_is_process_running(existing_pid)) {
        return true; // Process is running
    } else {
        // Process is dead, remove stale PID file
        LOG_WARN_MSG("Removing stale PID file for dead process %d", (int)existing_pid);
#ifdef PLATFORM_WINDOWS
        DeleteFileA(pid_file);
#else
        unlink(pid_file);
#endif
        return false;
    }
}

