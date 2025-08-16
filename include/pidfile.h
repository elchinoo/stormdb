#ifndef PIDFILE_H
#define PIDFILE_H

#include "stormdb.h"

// PID file management
bool pidfile_create(const char *pid_file);
void pidfile_remove(void);
bool pidfile_check_running(const char *pid_file);

#endif // PIDFILE_H
