#include "idx.h"

#include <stdlib.h>
#include <string.h>

struct idmix_idx {
    int max_objects;
    int max_variants;
    int check_bits;
    int check_mask;
};

static const int SW_BYTES[] = {1, 2, 4, 8};
static const int EMBEDDED_OTYPE[2][4] = {
    {IDMIX_OTYPE_UINT8, IDMIX_OTYPE_UINT16, IDMIX_OTYPE_UINT32, IDMIX_OTYPE_UINT64},
    {IDMIX_OTYPE_INT8, IDMIX_OTYPE_INT16, IDMIX_OTYPE_INT32, IDMIX_OTYPE_INT64},
};

typedef struct {
    uint8_t* data;
    size_t len;
} byte_buf_t;

typedef struct {
    int is_string;
    int otype;
    int64_t val;
    uint8_t* str;
    size_t str_len;
} data_object_t;

static void byte_buf_free(byte_buf_t* b) {
    free(b->data);
    b->data = NULL;
    b->len = 0;
}

static int byte_buf_init(byte_buf_t* b, size_t cap) {
    b->data = (uint8_t*)malloc(cap);
    b->len = 0;
    return b->data ? 0 : -1;
}

static int byte_buf_append(byte_buf_t* b, const uint8_t* data, size_t n) {
    uint8_t* nd = (uint8_t*)realloc(b->data, b->len + n);
    if (!nd) return -1;
    b->data = nd;
    memcpy(b->data + b->len, data, n);
    b->len += n;
    return 0;
}

static int is_unsigned(int otype) { return otype <= IDMIX_OTYPE_UINT64; }

static int width_bits(int otype) {
    switch (otype) {
        case IDMIX_OTYPE_UINT8:
        case IDMIX_OTYPE_INT8:
            return 0;
        case IDMIX_OTYPE_UINT16:
        case IDMIX_OTYPE_INT16:
            return 1;
        case IDMIX_OTYPE_UINT32:
        case IDMIX_OTYPE_INT32:
            return 2;
        default:
            return 3;
    }
}

static int validate_range(int otype, int64_t val) {
    switch (otype) {
        case IDMIX_OTYPE_UINT8:
            return val >= 0 && val <= 0xFF;
        case IDMIX_OTYPE_UINT16:
            return val >= 0 && val <= 0xFFFF;
        case IDMIX_OTYPE_UINT32:
            return val >= 0 && val <= 0xFFFFFFFFLL;
        case IDMIX_OTYPE_UINT64:
            return 1;
        case IDMIX_OTYPE_INT8:
            return val >= -128 && val <= 127;
        case IDMIX_OTYPE_INT16:
            return val >= -32768 && val <= 32767;
        case IDMIX_OTYPE_INT32:
            return val >= (-2147483647 - 1) && val <= 2147483647;
        case IDMIX_OTYPE_INT64:
            return 1;
        default:
            return 0;
    }
}

static int uint_to_le_bytes(uint64_t v, int size, uint8_t* out) {
    for (int i = 0; i < size; i++) out[i] = (uint8_t)((v >> (8 * i)) & 0xFF);
    return size;
}

static uint64_t le_bytes_to_uint(const uint8_t* payload, int size) {
    uint64_t u = 0;
    for (int i = 0; i < size; i++) u |= (uint64_t)payload[i] << (8 * i);
    return u;
}

static int64_t le_bytes_to_signed(const uint8_t* payload, int size) {
    uint64_t u = le_bytes_to_uint(payload, size);
    int shift = 64 - size * 8;
    return (int64_t)(u << shift) >> shift;
}

static int sw_from_magnitude(uint64_t mag) {
    if (mag < 256) return 0;
    if (mag < 65536) return 1;
    if (mag < 4294967296ULL) return 2;
    return 3;
}

static int sw_from_signed_value(int64_t val) {
    if (val >= -128 && val <= 127) return 0;
    if (val >= -32768 && val <= 32767) return 1;
    if (val >= (-2147483647 - 1) && val <= 2147483647) return 2;
    return 3;
}

static int signed_to_le_bytes(int64_t val, int size, uint8_t* out) {
    uint64_t u = (uint64_t)val;
    for (int i = 0; i < size; i++) out[i] = (uint8_t)((u >> (8 * i)) & 0xFF);
    return size;
}

static void magnitude_from_typed(int otype, int64_t val, uint64_t* mag, int* neg) {
    if (is_unsigned(otype)) {
        *mag = (uint64_t)val;
        *neg = 0;
        return;
    }
    if (val < 0) {
        *mag = (uint64_t)(-(uint64_t)val);
        *neg = 1;
    } else {
        *mag = (uint64_t)val;
        *neg = 0;
    }
}

static int try_embedded_head(int otype, int64_t val, uint8_t* head) {
    int neg;
    uint64_t mag;
    magnitude_from_typed(otype, val, &mag, &neg);
    if (mag >= 17) return 0;
    int wb = width_bits(otype);
    if (mag == 16) {
        if (neg) {
            *head = (uint8_t)((1 << 6) | (wb << 4) | 15);
            return 1;
        }
        return 0;
    }
    if (neg) {
        *head = (uint8_t)((1 << 6) | (wb << 4) | (int)(mag - 1));
    } else {
        *head = (uint8_t)((wb << 4) | (int)mag);
    }
    return 1;
}

static int payload_for_number(int otype, int64_t val, int* sw, uint8_t* payload, int* payload_len) {
    if (otype == IDMIX_OTYPE_UINT64) {
        uint64_t mag = (uint64_t)val;
        *sw = sw_from_magnitude(mag);
        *payload_len = uint_to_le_bytes(mag, SW_BYTES[*sw], payload);
        return 0;
    }
    if (is_unsigned(otype)) {
        if (val < 0) return -1;
        uint64_t mag = (uint64_t)val;
        *sw = sw_from_magnitude(mag);
        *payload_len = uint_to_le_bytes(mag, SW_BYTES[*sw], payload);
        return 0;
    }
    *sw = sw_from_signed_value(val);
    *payload_len = signed_to_le_bytes(val, SW_BYTES[*sw], payload);
    return 0;
}

static int value_from_payload(int otype, const uint8_t* payload, int size, int64_t* out) {
    if (is_unsigned(otype)) {
        uint64_t mag = le_bytes_to_uint(payload, size);
        if (otype != IDMIX_OTYPE_UINT64 && mag > (uint64_t)INT64_MAX) return -1;
        *out = (int64_t)mag;
        return 0;
    }
    *out = le_bytes_to_signed(payload, size);
    return 0;
}

static int normalize_object(const idmix_value_t* v, data_object_t* obj) {
    if (v->kind == IDMIX_KIND_STRING) {
        if (!v->str || v->str_len == 0 || v->str_len > IDMIX_MAX_STRING_LEN) return -1;
        obj->is_string = 1;
        obj->str_len = v->str_len;
        obj->str = (uint8_t*)malloc(v->str_len);
        if (!obj->str) return -1;
        memcpy(obj->str, v->str, v->str_len);
        return 0;
    }
    if (v->kind != IDMIX_KIND_INT) return -1;
    if (!validate_range(v->otype, v->val)) return -1;
    obj->is_string = 0;
    obj->otype = v->otype;
    obj->val = v->val;
    return 0;
}

static void free_data_object(data_object_t* obj) {
    free(obj->str);
    obj->str = NULL;
}

static int encode_object(const data_object_t* obj, uint8_t* out, int* out_len) {
    if (obj->is_string) {
        if (obj->str_len < 1 || obj->str_len > IDMIX_MAX_STRING_LEN) return -1;
        out[0] = (uint8_t)(0xC0 | obj->str_len);
        memcpy(out + 1, obj->str, obj->str_len);
        *out_len = (int)(1 + obj->str_len);
        return 0;
    }
    if (!validate_range(obj->otype, obj->val)) return -1;
    uint8_t embedded;
    if (try_embedded_head(obj->otype, obj->val, &embedded)) {
        out[0] = embedded;
        *out_len = 1;
        return 0;
    }
    uint8_t payload[8];
    int sw, plen;
    if (payload_for_number(obj->otype, obj->val, &sw, payload, &plen) != 0) return -1;
    out[0] = (uint8_t)(0x80 | (sw << 4) | obj->otype);
    memcpy(out + 1, payload, (size_t)plen);
    *out_len = 1 + plen;
    return 0;
}

static int decode_object(const uint8_t* data, size_t len, size_t offset, data_object_t* obj, size_t* consumed) {
    if (offset >= len) return -1;
    uint8_t head = data[offset];
    if ((head & 0x80) == 0) {
        int sign = (head >> 6) & 1;
        int wb = (head >> 4) & 0x03;
        int v = head & 0x0F;
        obj->is_string = 0;
        obj->otype = EMBEDDED_OTYPE[sign][wb];
        obj->val = sign == 0 ? v : -v - 1LL;
        *consumed = 1;
        return 0;
    }
    if ((head & 0x40) != 0) {
        size_t n = head & 0x3F;
        if (n < 1 || n > IDMIX_MAX_STRING_LEN) return -1;
        if (len < offset + 1 + n) return -1;
        obj->is_string = 1;
        obj->str_len = n;
        obj->str = (uint8_t*)malloc(n);
        if (!obj->str) return -1;
        memcpy(obj->str, data + offset + 1, n);
        *consumed = 1 + n;
        return 0;
    }
    int sw = (head >> 4) & 0x03;
    int otype = head & 0x0F;
    if (otype > IDMIX_OTYPE_INT64) return -1;
    int num_bytes = SW_BYTES[sw];
    if (len < offset + 1 + (size_t)num_bytes) return -1;
    obj->is_string = 0;
    obj->otype = otype;
    if (value_from_payload(otype, data + offset + 1, num_bytes, &obj->val) != 0) return -1;
    if (!validate_range(otype, obj->val)) return -1;
    *consumed = (size_t)(1 + num_bytes);
    return 0;
}

static int materialize_value(const data_object_t* obj, idmix_value_t* out) {
    if (obj->is_string) {
        out->kind = IDMIX_KIND_STRING;
        out->otype = 0;
        out->val = 0;
        out->str = (char*)malloc(obj->str_len + 1);
        if (!out->str) return -1;
        memcpy(out->str, obj->str, obj->str_len);
        out->str[obj->str_len] = '\0';
        out->str_len = obj->str_len;
        return 0;
    }
    out->kind = IDMIX_KIND_INT;
    out->otype = obj->otype;
    out->val = obj->val;
    out->str = NULL;
    out->str_len = 0;
    return 0;
}

idmix_idx_t* idmix_idx_create(int max_objects, int max_variants, int check_bits) {
    if (max_objects < 1 || max_objects > 255) return NULL;
    if (max_variants < 1 || max_variants > 32) return NULL;
    if (check_bits < 1 || check_bits > 2) return NULL;
    idmix_idx_t* idx = (idmix_idx_t*)calloc(1, sizeof(idmix_idx_t));
    if (!idx) return NULL;
    idx->max_objects = max_objects;
    idx->max_variants = max_variants;
    idx->check_bits = check_bits;
    idx->check_mask = (1 << check_bits) - 1;
    return idx;
}

void idmix_idx_destroy(idmix_idx_t* idx) { free(idx); }

int idmix_idx_max_objects(const idmix_idx_t* idx) { return idx ? idx->max_objects : 0; }
int idmix_idx_max_variants(const idmix_idx_t* idx) { return idx ? idx->max_variants : 0; }
int idmix_idx_check_bits(const idmix_idx_t* idx) { return idx ? idx->check_bits : 0; }

static int encode_binary(idmix_idx_t* idx, const idmix_value_t* values, size_t count, int variant_id,
                         uint8_t** out_data, size_t* out_len) {
    if (!idx || !values || count == 0 || !out_data || !out_len) return -1;
    if ((int)count > idx->max_objects) return -1;
    if (variant_id < 0 || variant_id >= idx->max_variants) return -1;

    data_object_t* objects = (data_object_t*)calloc(count, sizeof(data_object_t));
    if (!objects) return -1;
    for (size_t i = 0; i < count; i++) {
        if (normalize_object(&values[i], &objects[i]) != 0) {
            for (size_t j = 0; j <= i; j++) free_data_object(&objects[j]);
            free(objects);
            return -1;
        }
    }

    byte_buf_t obj_bytes;
    if (byte_buf_init(&obj_bytes, count * 9) != 0) {
        for (size_t i = 0; i < count; i++) free_data_object(&objects[i]);
        free(objects);
        return -1;
    }
    uint8_t ob[1 + IDMIX_MAX_STRING_LEN];
    int ob_len;
    for (size_t i = 0; i < count; i++) {
        if (encode_object(&objects[i], ob, &ob_len) != 0 ||
            byte_buf_append(&obj_bytes, ob, (size_t)ob_len) != 0) {
            byte_buf_free(&obj_bytes);
            for (size_t j = 0; j < count; j++) free_data_object(&objects[j]);
            free(objects);
            return -1;
        }
    }
    for (size_t i = 0; i < count; i++) free_data_object(&objects[i]);
    free(objects);

    uint8_t mask = (uint8_t)((variant_id * 0x9D + 0x37) & 0xFF);
    for (size_t i = 0; i < obj_bytes.len; i++) obj_bytes.data[i] ^= mask;

    int header_len = count == 1 ? 1 : 2;
    size_t total = (size_t)header_len + obj_bytes.len;
    uint8_t* data = (uint8_t*)malloc(total);
    if (!data) {
        byte_buf_free(&obj_bytes);
        return -1;
    }
    if (count == 1) {
        data[0] = (uint8_t)(variant_id << idx->check_bits);
    } else {
        data[0] = (uint8_t)(0x80 | (variant_id << idx->check_bits));
        data[1] = (uint8_t)count;
    }
    memcpy(data + header_len, obj_bytes.data, obj_bytes.len);
    byte_buf_free(&obj_bytes);

    int xor_sum = 0;
    for (size_t i = 0; i < total; i++) xor_sum ^= data[i];
    data[0] |= (uint8_t)(xor_sum & idx->check_mask);

    *out_data = data;
    *out_len = total;
    return 0;
}

static int decode_binary(idmix_idx_t* idx, const uint8_t* data, size_t data_len, idmix_value_t** out_values,
                         size_t* out_count) {
    if (!idx || !data || data_len < 1 || !out_values || !out_count) return -1;

    uint8_t byte0 = data[0];
    uint8_t check = (uint8_t)(byte0 & idx->check_mask);
    int multi = (byte0 & 0x80) != 0;
    int variant_id = (byte0 & 0x7F) >> idx->check_bits;

    if (variant_id >= idx->max_variants) return -1;

    int header_len = 1;
    int count = 1;
    if (multi) {
        if (data_len < 2) return -1;
        header_len = 2;
        count = data[1];
        if (count < 2 || count > idx->max_objects) return -1;
    }

    uint8_t* verify = (uint8_t*)malloc(data_len);
    if (!verify) return -1;
    memcpy(verify, data, data_len);
    verify[0] &= (uint8_t)~idx->check_mask;
    int xor_sum = 0;
    for (size_t i = 0; i < data_len; i++) xor_sum ^= verify[i];
    free(verify);
    if ((xor_sum & idx->check_mask) != check) return -1;

    if (data_len < (size_t)header_len) return -1;
    size_t obj_len = data_len - (size_t)header_len;
    uint8_t* obj_data = (uint8_t*)malloc(obj_len);
    if (!obj_data) return -1;
    memcpy(obj_data, data + header_len, obj_len);
    uint8_t mask = (uint8_t)((variant_id * 0x9D + 0x37) & 0xFF);
    for (size_t i = 0; i < obj_len; i++) obj_data[i] ^= mask;

    idmix_value_t* result = (idmix_value_t*)calloc((size_t)count, sizeof(idmix_value_t));
    if (!result) {
        free(obj_data);
        return -1;
    }
    size_t pos = 0;
    for (int i = 0; i < count; i++) {
        data_object_t obj = {0};
        size_t consumed = 0;
        if (decode_object(obj_data, obj_len, pos, &obj, &consumed) != 0 ||
            materialize_value(&obj, &result[i]) != 0) {
            free_data_object(&obj);
            free(obj_data);
            for (int j = 0; j < i; j++) free(result[j].str);
            free(result);
            return -1;
        }
        free_data_object(&obj);
        pos += consumed;
    }
    free(obj_data);
    if (pos != obj_len) {
        for (int i = 0; i < count; i++) free(result[i].str);
        free(result);
        return -1;
    }
    *out_values = result;
    *out_count = (size_t)count;
    return 0;
}

int idmix_idx_encode(idmix_idx_t* idx, const idmix_value_t* values, size_t count, uint8_t** out_data,
                     size_t* out_len) {
    return encode_binary(idx, values, count, 0, out_data, out_len);
}

int idmix_idx_encode_with_variant(idmix_idx_t* idx, int variant_id, const idmix_value_t* values, size_t count,
                                  uint8_t** out_data, size_t* out_len) {
    return encode_binary(idx, values, count, variant_id, out_data, out_len);
}

int idmix_idx_decode(idmix_idx_t* idx, const uint8_t* data, size_t data_len, idmix_value_t** out_values,
                     size_t* out_count) {
    (void)idx;
    return decode_binary(idx, data, data_len, out_values, out_count);
}

void idmix_idx_free_bytes(uint8_t* data) { free(data); }

void idmix_idx_free_values(idmix_value_t* values, size_t count) {
    if (!values) return;
    for (size_t i = 0; i < count; i++) free(values[i].str);
    free(values);
}
