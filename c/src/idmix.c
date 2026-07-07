#include "idmix.h"

#include <stdlib.h>
#include <time.h>

struct idmix_ctx {
    idmix_idx_t* idx;
    idmix_codec_t* codec;
};

idmix_ctx_t* idmix_create(const char* alphabet) {
    idmix_ctx_t* ctx = (idmix_ctx_t*)calloc(1, sizeof(idmix_ctx_t));
    if (!ctx) return NULL;
    ctx->idx = idmix_idx_create(255, 32, 2);
    ctx->codec = idmix_codec_create(alphabet);
    if (!ctx->idx || !ctx->codec) {
        idmix_destroy(ctx);
        return NULL;
    }
    return ctx;
}

void idmix_destroy(idmix_ctx_t* ctx) {
    if (!ctx) return;
    idmix_idx_destroy(ctx->idx);
    idmix_codec_destroy(ctx->codec);
    free(ctx);
}

idmix_idx_t* idmix_get_idx(idmix_ctx_t* ctx) { return ctx ? ctx->idx : NULL; }
idmix_codec_t* idmix_get_codec(idmix_ctx_t* ctx) { return ctx ? ctx->codec : NULL; }

void idmix_free_string(char* s) { idmix_codec_free_string(s); }

void idmix_free_values(idmix_value_t* values, size_t count) { idmix_idx_free_values(values, count); }

int idmix_encode_with_variant(idmix_ctx_t* ctx, int variant_id, const idmix_value_t* values, size_t count,
                              char** out) {
    if (!ctx || !values || count == 0 || !out) return -1;
    uint8_t* data = NULL;
    size_t data_len = 0;
    if (idmix_idx_encode_with_variant(ctx->idx, variant_id, values, count, &data, &data_len) != 0) return -1;
    int rc = idmix_codec_encode_bytes(ctx->codec, data, data_len, out);
    idmix_idx_free_bytes(data);
    return rc;
}

int idmix_encode(idmix_ctx_t* ctx, const idmix_value_t* values, size_t count, char** out) {
    if (!ctx || !values || count == 0 || !out) return -1;
    int variant_id = (int)(rand() % idmix_idx_max_variants(ctx->idx));
    return idmix_encode_with_variant(ctx, variant_id, values, count, out);
}

int idmix_decode(idmix_ctx_t* ctx, const char* s, idmix_value_t** values, size_t* count) {
    if (!ctx || !s || !values || !count) return -1;
    uint8_t* data = NULL;
    size_t data_len = 0;
    if (idmix_codec_decode_bytes(ctx->codec, s, &data, &data_len) != 0) return -1;
    int rc = idmix_idx_decode(ctx->idx, data, data_len, values, count);
    idmix_codec_free_bytes(data);
    return rc;
}
