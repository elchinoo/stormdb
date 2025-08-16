#ifndef STORMDB_API_H
#define STORMDB_API_H

#include "stormdb.h"
#include "config.h"
#include <stdbool.h>

bool api_start(int port);
void api_stop(void);
bool api_restart(int new_port);

#endif // STORMDB_API_H
