//go:build darwin && cgo

#include "interface_observer_darwin.h"
#include <SystemConfiguration/SystemConfiguration.h>
#include <dispatch/dispatch.h>
#include <stdlib.h>

extern void goInterfaceChanged(uintptr_t handle);

struct SPInterfaceObserver {
    SCDynamicStoreRef store;
    dispatch_queue_t queue;
};

static void changed(SCDynamicStoreRef store, CFArrayRef keys, void *info) {
    goInterfaceChanged((uintptr_t)info);
}

static void drain(void *context) {}

void sp_interface_observer_stop(SPInterfaceObserver *observer) {
    if (!observer) return;
    if (observer->store) {
        SCDynamicStoreSetDispatchQueue(observer->store, NULL);
    }
    // Wait for callbacks before Go releases the callback handle.
    if (observer->queue) dispatch_sync_f(observer->queue, NULL, drain);
    if (observer->store) CFRelease(observer->store);
    if (observer->queue) dispatch_release(observer->queue);
    free(observer);
}

SPInterfaceObserver *sp_interface_observer_start(uintptr_t handle) {
    SPInterfaceObserver *observer = calloc(1, sizeof(*observer));
    if (!observer) return NULL;
    SCDynamicStoreContext context = {0, (void *)handle, NULL, NULL, NULL};
    observer->store = SCDynamicStoreCreate(NULL, CFSTR("SmartProxy interfaces"), changed, &context);
    if (!observer->store) goto fail;
    const void *patterns[] = {
        CFSTR("^State:/Network/Interface$"),
        CFSTR("^State:/Network/Interface/[^/]+/(IPv4|IPv6|Link)$"),
        CFSTR("^State:/Network/Service/[^/]+/(IPv4|IPv6)$"),
        CFSTR("^State:/Network/Global/(IPv4|IPv6)$")
    };
    CFArrayRef subscriptions = CFArrayCreate(NULL, patterns, 4, &kCFTypeArrayCallBacks);
    if (!subscriptions) goto fail;
    Boolean ok = SCDynamicStoreSetNotificationKeys(observer->store, NULL, subscriptions);
    CFRelease(subscriptions);
    if (!ok) goto fail;
    observer->queue = dispatch_queue_create("smartproxy.interfaces", DISPATCH_QUEUE_SERIAL);
    if (!observer->queue || !SCDynamicStoreSetDispatchQueue(observer->store, observer->queue)) goto fail;
    return observer;
fail:
    sp_interface_observer_stop(observer);
    return NULL;
}
