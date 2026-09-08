#include <stdint.h>
typedef struct SPInterfaceObserver SPInterfaceObserver;
SPInterfaceObserver *sp_interface_observer_start(uintptr_t handle);
void sp_interface_observer_stop(SPInterfaceObserver *observer);
