#ifndef IDMIX_H
#define IDMIX_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define IDMIX_DEFAULT_ALPHABET "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

#define IDMIX_OTYPE_UINT8  0
#define IDMIX_OTYPE_UINT16 1
#define IDMIX_OTYPE_UINT32 2
#define IDMIX_OTYPE_UINT64 3
#define IDMIX_OTYPE_INT8   4
#define IDMIX_OTYPE_INT16  5
#define IDMIX_OTYPE_INT32  6
#define IDMIX_OTYPE_INT64  7

typedef struct {
    int otype;
    int64_t val;
} idmix_value_t;

typedef struct idmix_ctx idmix_ctx_t;

/* Create codec; alphabet=NULL uses default. Returns NULL on failure. */
idmix_ctx_t* idmix_create(const char* alphabet);
void idmix_destroy(idmix_ctx_t* ctx);

/* Encode typed values to XID string. *out is malloc'd; call idmix_free_string. */
int idmix_encode(idmix_ctx_t* ctx, const idmix_value_t* values, size_t count, char** out);

/* Decode XID string. *values and *count are malloc'd; call idmix_free_values. */
int idmix_decode(idmix_ctx_t* ctx, const char* s, idmix_value_t** values, size_t* count);

void idmix_free_string(char* s);
void idmix_free_values(idmix_value_t* values);

#ifdef __cplusplus
}
#endif

#endif /* IDMIX_H */
