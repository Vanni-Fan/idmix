#ifndef IDMIX_H
#define IDMIX_H

#include "codec.h"
#include "idx.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct idmix_ctx idmix_ctx_t;

/* Create IdMix (default Idx + RadixCodec). alphabet=NULL uses default. */
idmix_ctx_t* idmix_create(const char* alphabet);
void idmix_destroy(idmix_ctx_t* ctx);

idmix_idx_t* idmix_get_idx(idmix_ctx_t* ctx);
idmix_codec_t* idmix_get_codec(idmix_ctx_t* ctx);

/* Encode typed values or strings to text. *out is malloc'd; call idmix_free_string. */
int idmix_encode(idmix_ctx_t* ctx, const idmix_value_t* values, size_t count, char** out);
int idmix_encode_with_variant(idmix_ctx_t* ctx, int variant_id, const idmix_value_t* values, size_t count,
                              char** out);

/* Decode text. *values and *count are malloc'd; call idmix_free_values. */
int idmix_decode(idmix_ctx_t* ctx, const char* s, idmix_value_t** values, size_t* count);

void idmix_free_string(char* s);
void idmix_free_values(idmix_value_t* values, size_t count);

#ifdef __cplusplus
}
#endif

#endif /* IDMIX_H */
