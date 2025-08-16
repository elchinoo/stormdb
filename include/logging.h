#ifndef LOGGING_H
#define LOGGING_H

#include "stormdb.h"

/** Log severity levels. */
typedef enum {
    LOG_ERROR = 0,
    LOG_WARN = 1,
    LOG_INFO = 2,
    LOG_DEBUG = 3,
    LOG_TRACE = 4
} log_level_t;

/** Initialize logging with a given level. Must be called before other logging calls. */
bool logging_init(log_level_t level);
/** Flush and close logging sinks; safe to call multiple times. */
void logging_cleanup(void);
/** Change the active log level at runtime (e.g., on reload). */
void logging_set_level(log_level_t level);
/** Set the logfile path (creates/truncates as needed). Returns false on failure. */
bool logging_set_file(const char *filepath);
/** Configure size-based rotation.
 *  When the file exceeds max_size bytes, rotate keeping up to max_files.
 *  If max_size == 0 or max_files <= 0, rotation is disabled. */
void logging_set_rotation(size_t max_size, int max_files);
/** Core logging call with printf-style formatting and source location. */
void log_message(log_level_t level, const char *file, int line, const char *func, const char *format, ...) __attribute__((format(printf, 5, 6)));

// Convenience macros
#define LOG_ERROR_MSG(...) log_message(LOG_ERROR, __FILE__, __LINE__, __func__, __VA_ARGS__)
#define LOG_WARN_MSG(...)  log_message(LOG_WARN, __FILE__, __LINE__, __func__, __VA_ARGS__)
#define LOG_INFO_MSG(...)   log_message(LOG_INFO, __FILE__, __LINE__, __func__, __VA_ARGS__)
#define LOG_DEBUG_MSG(...) log_message(LOG_DEBUG, __FILE__, __LINE__, __func__, __VA_ARGS__)
#define LOG_TRACE_MSG(...) log_message(LOG_TRACE, __FILE__, __LINE__, __func__, __VA_ARGS__)

#endif // LOGGING_H
