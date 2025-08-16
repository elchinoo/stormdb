#include "logging.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <stdarg.h>
#include <pthread.h>
#include <sys/time.h>

static log_level_t current_log_level = LOG_INFO;
static FILE *log_file = NULL;
static pthread_mutex_t log_mutex = PTHREAD_MUTEX_INITIALIZER;
static size_t rotate_max_size = 0; // bytes; 0=disabled
static int rotate_max_files = 0;   // <=0=disabled
static char current_log_path[1024] = {0};

static const char* log_level_strings[] = {
    "ERROR", "WARN", "INFO", "DEBUG", "TRACE"
};

bool logging_init(log_level_t level) {
    current_log_level = level;
    log_file = stderr; // Default to stderr
    
    printf("Logging initialized with level: %s\n", log_level_strings[level]);
    return true;
}

void logging_cleanup(void) {
    pthread_mutex_lock(&log_mutex);
    
    if (log_file && log_file != stderr && log_file != stdout) {
        fclose(log_file);
    }
    log_file = NULL;
    
    pthread_mutex_unlock(&log_mutex);
}

bool logging_set_file(const char *filename) {
    if (!filename) {
        return false;
    }
    
    pthread_mutex_lock(&log_mutex);
    
    FILE *new_file = fopen(filename, "a");
    if (!new_file) {
        pthread_mutex_unlock(&log_mutex);
        return false;
    }
    
    if (log_file && log_file != stderr && log_file != stdout) {
        fclose(log_file);
    }
    
    log_file = new_file;
    strncpy(current_log_path, filename, sizeof(current_log_path)-1);
    pthread_mutex_unlock(&log_mutex);
    
    return true;
}

void logging_set_level(log_level_t level) {
    current_log_level = level;
}

void logging_set_rotation(size_t max_size, int max_files) {
    pthread_mutex_lock(&log_mutex);
    rotate_max_size = max_size;
    rotate_max_files = max_files;
    pthread_mutex_unlock(&log_mutex);
}

static void rotate_logs_if_needed_unlocked(void) {
    if (!log_file || current_log_path[0] == '\0') return;
    if (rotate_max_size == 0 || rotate_max_files <= 0) return;
    long pos = ftell(log_file);
    if (pos < 0) return;
    if ((size_t)pos < rotate_max_size) return;

    // Close current file before rotation
    if (log_file != stderr && log_file != stdout) {
        fclose(log_file);
    }
    // Rotate: file.(N-1)->file.N ... file.1->file.2 ; file->file.1
    for (int i = rotate_max_files - 1; i >= 1; --i) {
        char src[1200], dst[1200];
        snprintf(src, sizeof(src), "%s.%d", current_log_path, i);
        snprintf(dst, sizeof(dst), "%s.%d", current_log_path, i+1);
        rename(src, dst); // ignore errors
    }
    char first[1200];
    snprintf(first, sizeof(first), "%s.1", current_log_path);
    rename(current_log_path, first);
    // Reopen fresh log file
    log_file = fopen(current_log_path, "a");
    if (!log_file) {
        log_file = stderr; // fallback
    }
}

void log_message(log_level_t level, const char *file, int line, const char *func, const char *format, ...) {
    if (level > current_log_level) {
        return;
    }
    
    pthread_mutex_lock(&log_mutex);
    
    if (!log_file) {
        log_file = stderr;
    }
    
    // Get current time
    struct timeval tv;
    gettimeofday(&tv, NULL);
    struct tm *tm_info = localtime(&tv.tv_sec);
    
    // Format timestamp
    char timestamp[64];
    strftime(timestamp, sizeof(timestamp), "%Y-%m-%d %H:%M:%S", tm_info);
    
    // Print log message
    fprintf(log_file, "[%s.%03d] [%s] [%s:%d:%s] ", 
            timestamp, (int)(tv.tv_usec / 1000),
            log_level_strings[level], file, line, func);
    
    va_list args;
    va_start(args, format);
    vfprintf(log_file, format, args);
    va_end(args);
    
    fprintf(log_file, "\n");
    fflush(log_file);
    // Rotate after each write to keep it simple
    rotate_logs_if_needed_unlocked();
    
    pthread_mutex_unlock(&log_mutex);
}
