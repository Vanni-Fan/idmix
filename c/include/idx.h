#ifndef IDMIX_IDX_H
#define IDMIX_IDX_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define IDMIX_MAX_STRING_LEN 63

#define IDMIX_OTYPE_UINT8  0
#define IDMIX_OTYPE_UINT16 1
#define IDMIX_OTYPE_UINT32 2
#define IDMIX_OTYPE_UINT64 3
#define IDMIX_OTYPE_INT8   4
#define IDMIX_OTYPE_INT16  5
#define IDMIX_OTYPE_INT32  6
#define IDMIX_OTYPE_INT64  7

#define IDMIX_KIND_INT    0
#define IDMIX_KIND_STRING 1

typedef struct {
    int kind;
    int otype;
    int64_t val;
    char* str;
    size_t str_len;
} idmix_value_t;

#define IDMIX_INT(otype, val) \
    ((idmix_value_t){IDMIX_KIND_INT, (otype), (int64_t)(val), NULL, 0})

typedef struct idmix_idx idmix_idx_t;

idmix_idx_t* idmix_idx_create(int max_objects, int max_variants, int check_bits);
void idmix_idx_destroy(idmix_idx_t* idx);

int idmix_idx_max_objects(const idmix_idx_t* idx);
int idmix_idx_max_variants(const idmix_idx_t* idx);
int idmix_idx_check_bits(const idmix_idx_t* idx);

/* *out_data is malloc'd; call idmix_idx_free_bytes. */
int idmix_idx_encode(idmix_idx_t* idx, const idmix_value_t* values, size_t count, uint8_t** out_data,
                     size_t* out_len);
int idmix_idx_encode_with_variant(idmix_idx_t* idx, int variant_id, const idmix_value_t* values,
                                  size_t count, uint8_t** out_data, size_t* out_len);

/* *out_values is malloc'd; string fields are malloc'd; call idmix_idx_free_values. */
int idmix_idx_decode(idmix_idx_t* idx, const uint8_t* data, size_t data_len, idmix_value_t** out_values,
                     size_t* out_count);

void idmix_idx_free_bytes(uint8_t* data);
void idmix_idx_free_values(idmix_value_t* values, size_t count);

#ifdef __cplusplus
}
#endif

#endif /* IDMIX_IDX_H */
