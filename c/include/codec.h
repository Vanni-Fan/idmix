#ifndef IDMIX_CODEC_H
#define IDMIX_CODEC_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define IDMIX_DEFAULT_ALPHABET "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

typedef struct idmix_codec idmix_codec_t;

idmix_codec_t* idmix_codec_create(const char* alphabet);
void idmix_codec_destroy(idmix_codec_t* codec);

/* *out is malloc'd; call idmix_codec_free_string. */
int idmix_codec_encode_bytes(idmix_codec_t* codec, const uint8_t* data, size_t data_len, char** out);
int idmix_codec_decode_bytes(idmix_codec_t* codec, const char* s, uint8_t** out, size_t* out_len);

void idmix_codec_free_string(char* s);
void idmix_codec_free_bytes(uint8_t* data);

/* Standalone helpers using the default alphabet. */
int idmix_encode_bytes(const uint8_t* data, size_t data_len, char** out);
int idmix_decode_string(const char* s, uint8_t** out, size_t* out_len);

#ifdef __cplusplus
}
#endif

#endif /* IDMIX_CODEC_H */
