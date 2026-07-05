#include "idmix.h"

#include <stdlib.h>
#include <string.h>
#include <time.h>

struct idmix_ctx {
    char* alphabet;
    int base;
    int* from_custom;
    int max_objects;
    int max_variants;
    int check_bits;
    int count_bits;
    int variant_bits;
    int check_mask;
    int count_mask;
    int variant_mask;
    int count_shift;
    int variant_shift;
};

typedef struct {
    uint8_t* data;
    size_t len;
} byte_buf_t;

static int bit_len(int n) {
    if (n <= 0) return 1;
    int bits = 0;
    while (n > 0) {
        n >>= 1;
        bits++;
    }
    return bits;
}

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

static idmix_ctx_t* ctx_finalize(idmix_ctx_t* ctx) {
    int variant_bits = ctx->max_variants <= 1 ? 1 : bit_len(ctx->max_variants - 1);
    int count_bits = ctx->max_objects <= 1 ? 1 : bit_len(ctx->max_objects);
    int total = ctx->check_bits + count_bits + variant_bits;
    if (total > 16) {
        idmix_destroy(ctx);
        return NULL;
    }
    ctx->count_bits = count_bits;
    ctx->variant_bits = variant_bits;
    ctx->check_mask = (1 << ctx->check_bits) - 1;
    ctx->count_mask = ((1 << count_bits) - 1) << ctx->check_bits;
    ctx->variant_mask = ((1 << variant_bits) - 1) << (ctx->check_bits + count_bits);
    ctx->count_shift = ctx->check_bits;
    ctx->variant_shift = ctx->check_bits + count_bits;
    return ctx;
}

idmix_ctx_t* idmix_create(const char* alphabet) {
    if (!alphabet) alphabet = IDMIX_DEFAULT_ALPHABET;
    size_t len = strlen(alphabet);
    if (len < 2) return NULL;

    idmix_ctx_t* ctx = (idmix_ctx_t*)calloc(1, sizeof(idmix_ctx_t));
    if (!ctx) return NULL;
    ctx->alphabet = (char*)malloc(len + 1);
    ctx->from_custom = (int*)malloc(256 * sizeof(int));
    if (!ctx->alphabet || !ctx->from_custom) {
        idmix_destroy(ctx);
        return NULL;
    }
    memcpy(ctx->alphabet, alphabet, len + 1);
    ctx->base = (int)len;
    for (int i = 0; i < 256; i++) ctx->from_custom[i] = -1;
    for (size_t i = 0; i < len; i++) {
        unsigned char c = (unsigned char)alphabet[i];
        if (ctx->from_custom[c] >= 0) {
            idmix_destroy(ctx);
            return NULL;
        }
        ctx->from_custom[c] = (int)i;
    }
    ctx->max_objects = 511;
    ctx->max_variants = 32;
    ctx->check_bits = 2;
    return ctx_finalize(ctx);
}

void idmix_destroy(idmix_ctx_t* ctx) {
    if (!ctx) return;
    free(ctx->alphabet);
    free(ctx->from_custom);
    free(ctx);
}

void idmix_free_string(char* s) { free(s); }
void idmix_free_values(idmix_value_t* values) { free(values); }

typedef struct {
    uint8_t* data;
    size_t len;
} bigint_t;

static void bigint_free(bigint_t* n) {
    free(n->data);
    n->data = NULL;
    n->len = 0;
}

static int bigint_from_bytes(bigint_t* n, const uint8_t* bytes, size_t len) {
    n->data = (uint8_t*)malloc(len ? len : 1);
    if (!n->data) return -1;
    n->len = len ? len : 1;
    if (len) memcpy(n->data, bytes, len);
    else n->data[0] = 0;
    return 0;
}

static void bigint_trim(bigint_t* n) {
    while (n->len > 1 && n->data[0] == 0) {
        memmove(n->data, n->data + 1, n->len - 1);
        n->len--;
    }
}

static int bigint_divmod_small(const bigint_t* n, int base, int* rem, bigint_t* quot) {
    int carry = 0;
    quot->data = (uint8_t*)malloc(n->len);
    if (!quot->data) return -1;
    quot->len = n->len;
    for (size_t i = 0; i < n->len; i++) {
        int cur = carry * 256 + n->data[i];
        quot->data[i] = (uint8_t)(cur / base);
        carry = cur % base;
    }
    bigint_trim(quot);
    *rem = carry;
    return 0;
}

static int bigint_copy(bigint_t* dst, const bigint_t* src) {
    bigint_free(dst);
    return bigint_from_bytes(dst, src->data, src->len);
}

static int bigint_muladd_small(bigint_t* n, int base, int add) {
    int carry = add;
    for (size_t i = n->len; i > 0; i--) {
        int cur = n->data[i - 1] * base + carry;
        n->data[i - 1] = (uint8_t)(cur & 0xFF);
        carry = cur >> 8;
    }
    while (carry > 0) {
        uint8_t* nd = (uint8_t*)realloc(n->data, n->len + 1);
        if (!nd) return -1;
        n->data = nd;
        memmove(n->data + 1, n->data, n->len);
        n->data[0] = (uint8_t)(carry & 0xFF);
        n->len++;
        carry >>= 8;
    }
    bigint_trim(n);
    return 0;
}

static char* radix_encode_bytes(idmix_ctx_t* ctx, const uint8_t* data, size_t data_len) {
    if (data_len == 0) {
        char* s = (char*)malloc(2);
        if (!s) return NULL;
        s[0] = ctx->alphabet[0];
        s[1] = '\0';
        return s;
    }
    uint8_t* wrapped = (uint8_t*)malloc(2 + data_len);
    if (!wrapped) return NULL;
    wrapped[0] = (uint8_t)((data_len >> 8) & 0xFF);
    wrapped[1] = (uint8_t)(data_len & 0xFF);
    memcpy(wrapped + 2, data, data_len);

    bigint_t n, q;
    bigint_from_bytes(&n, wrapped, 2 + data_len);
    free(wrapped);
    bigint_t zero;
    bigint_from_bytes(&zero, (const uint8_t*)"\0", 1);

    char rev[512];
    int rlen = 0;
    while (n.len > 1 || n.data[0] != 0) {
        int rem = 0;
        if (bigint_divmod_small(&n, ctx->base, &rem, &q) != 0) {
            bigint_free(&n);
            bigint_free(&q);
            bigint_free(&zero);
            return NULL;
        }
        if (rlen >= (int)sizeof(rev) - 1) {
            bigint_free(&n);
            bigint_free(&q);
            bigint_free(&zero);
            return NULL;
        }
        rev[rlen++] = ctx->alphabet[rem];
        bigint_free(&n);
        bigint_from_bytes(&n, q.data, q.len);
        bigint_free(&q);
        bigint_from_bytes(&q, zero.data, zero.len);
    }
    bigint_free(&n);
    bigint_free(&q);
    bigint_free(&zero);

    char* out = (char*)malloc((size_t)rlen + 1);
    if (!out) return NULL;
    for (int i = 0; i < rlen; i++) out[i] = rev[rlen - 1 - i];
    out[rlen] = '\0';
    return out;
}

static int radix_decode_bytes(idmix_ctx_t* ctx, const char* s, uint8_t** out, size_t* out_len) {
    if (!s || !s[0]) return -1;
    bigint_t n;
    bigint_from_bytes(&n, (const uint8_t*)"\0", 1);
    for (const char* p = s; *p; p++) {
        unsigned char c = (unsigned char)*p;
        if (ctx->from_custom[c] < 0) {
            bigint_free(&n);
            return -1;
        }
        if (bigint_muladd_small(&n, ctx->base, ctx->from_custom[c]) != 0) {
            bigint_free(&n);
            return -1;
        }
    }
    for (int pad = 0; pad <= 1; pad++) {
        size_t raw_len = n.len + (size_t)pad;
        uint8_t* buf = (uint8_t*)calloc(raw_len, 1);
        if (!buf) {
            bigint_free(&n);
            return -1;
        }
        if (pad) buf[0] = 0;
        memcpy(buf + pad, n.data, n.len);
        if (raw_len >= 2) {
            int data_len = (buf[0] << 8) | buf[1];
            if ((int)raw_len == 2 + data_len) {
                *out = (uint8_t*)malloc((size_t)data_len);
                if (!*out) {
                    free(buf);
                    bigint_free(&n);
                    return -1;
                }
                memcpy(*out, buf + 2, (size_t)data_len);
                *out_len = (size_t)data_len;
                free(buf);
                bigint_free(&n);
                return 0;
            }
        }
        free(buf);
    }
    bigint_free(&n);
    return -1;
}

/* xid binary layer */
static const int SW_BYTES[] = {1, 2, 4, 8};
static const int EMBEDDED_OTYPE[2][4] = {
    {IDMIX_OTYPE_UINT8, IDMIX_OTYPE_UINT16, IDMIX_OTYPE_UINT32, IDMIX_OTYPE_UINT64},
    {IDMIX_OTYPE_INT8, IDMIX_OTYPE_INT16, IDMIX_OTYPE_INT32, IDMIX_OTYPE_INT64},
};

static int is_unsigned(int otype) { return otype <= IDMIX_OTYPE_UINT64; }
static int is_signed(int otype) { return otype >= IDMIX_OTYPE_INT8; }

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

static int target_bits(int otype) {
    switch (otype) {
        case IDMIX_OTYPE_UINT8:
        case IDMIX_OTYPE_INT8:
            return 8;
        case IDMIX_OTYPE_UINT16:
        case IDMIX_OTYPE_INT16:
            return 16;
        case IDMIX_OTYPE_UINT32:
        case IDMIX_OTYPE_INT32:
            return 32;
        default:
            return 64;
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
            return val >= 0;
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

static int64_t reconstruct_int(int otype, int sw, uint64_t raw) {
    int tbits = target_bits(otype);
    int stored_bits = SW_BYTES[sw] * 8;
    if (is_unsigned(otype)) {
        uint64_t mask = tbits == 64 ? ~0ULL : ((1ULL << tbits) - 1);
        return (int64_t)(raw & mask);
    }
    int sign_bit = (int)((raw >> (stored_bits - 1)) & 1);
    if (tbits <= stored_bits) {
        uint64_t mask = (1ULL << tbits) - 1;
        int64_t val = (int64_t)(raw & mask);
        if (sign_bit == 1 && (val & (1LL << (tbits - 1)))) val -= 1LL << tbits;
        return val;
    }
    int64_t extended;
    if (sign_bit == 1) {
        uint64_t extend_mask = (~((1ULL << stored_bits) - 1)) & ((1ULL << tbits) - 1);
        extended = (int64_t)(raw | extend_mask);
    } else {
        extended = (int64_t)raw;
    }
    if (extended >= (1LL << (tbits - 1))) extended -= 1LL << tbits;
    return extended;
}

static int minimal_complement_bytes(int otype, int64_t val, int* sw_out, uint8_t* payload, int* payload_len) {
    if (val == 0) {
        *sw_out = 0;
        payload[0] = 0;
        *payload_len = 1;
        return 0;
    }
    if (is_unsigned(otype)) {
        if (val < 0) return -1;
        uint64_t uval = (uint64_t)val;
        for (int sw = 0; sw < 4; sw++) {
            int size = SW_BYTES[sw];
            if (size < 8 && uval >= (1ULL << (size * 8))) continue;
            uint_to_le_bytes(uval, size, payload);
            if ((payload[size - 1] & 0x80) == 0) {
                *sw_out = sw;
                *payload_len = size;
                return 0;
            }
        }
        return -1;
    }
    int tbits = target_bits(otype);
    uint64_t mask = tbits == 64 ? ~0ULL : ((1ULL << tbits) - 1);
    uint64_t uval = (uint64_t)val & mask;

    if (val < 0) {
        for (int sw = 0; sw < 4; sw++) {
            int size = SW_BYTES[sw];
            int shift = size * 8;
            if (shift >= tbits) {
                *sw_out = sw;
                *payload_len = uint_to_le_bytes(uval, size, payload);
                return 0;
            }
            uint64_t lower = uval & ((1ULL << shift) - 1);
            uint64_t upper = uval >> shift;
            uint64_t upper_mask = (1ULL << (tbits - shift)) - 1;
            if (upper != upper_mask) continue;
            int high_byte = (int)((lower >> (shift - 8)) & 0xFF);
            if ((high_byte & 0x80) == 0) continue;
            *sw_out = sw;
            *payload_len = uint_to_le_bytes(lower, size, payload);
            return 0;
        }
    } else {
        for (int sw = 0; sw < 4; sw++) {
            int size = SW_BYTES[sw];
            if (size < 8 && uval >= (1ULL << (size * 8))) continue;
            uint_to_le_bytes(uval, size, payload);
            if ((payload[size - 1] & 0x80) == 0) {
                *sw_out = sw;
                *payload_len = size;
                return 0;
            }
        }
    }
    int sw_final = 3;
    if (tbits == 8) sw_final = 0;
    else if (tbits == 16) sw_final = 1;
    else if (tbits == 32) sw_final = 2;
    *sw_out = sw_final;
    *payload_len = uint_to_le_bytes(uval, SW_BYTES[sw_final], payload);
    return 0;
}

static int encode_object(const idmix_value_t* tv, uint8_t* out, int* out_len) {
    if (!validate_range(tv->otype, tv->val)) return -1;
    if (is_unsigned(tv->otype) && tv->val >= 0 && tv->val <= 15) {
        int wb = width_bits(tv->otype);
        out[0] = (uint8_t)((wb << 4) | tv->val);
        *out_len = 1;
        return 0;
    }
    if (is_signed(tv->otype) && tv->val >= -16 && tv->val <= -1) {
        int wb = width_bits(tv->otype);
        int v = (int)(-tv->val - 1);
        out[0] = (uint8_t)((1 << 6) | (wb << 4) | v);
        *out_len = 1;
        return 0;
    }
    int sw, plen;
    uint8_t payload[8];
    if (minimal_complement_bytes(tv->otype, tv->val, &sw, payload, &plen) != 0) return -1;
    out[0] = (uint8_t)(0x80 | (sw << 4) | tv->otype);
    memcpy(out + 1, payload, (size_t)plen);
    *out_len = 1 + plen;
    return 0;
}

static int decode_object(const uint8_t* data, size_t len, size_t offset, idmix_value_t* tv, size_t* consumed) {
    if (offset >= len) return -1;
    uint8_t head = data[offset];
    if ((head & 0x80) == 0) {
        int sign = (head >> 6) & 1;
        int wb = (head >> 4) & 0x03;
        int v = head & 0x0F;
        tv->otype = EMBEDDED_OTYPE[sign][wb];
        tv->val = sign == 0 ? v : -v - 1LL;
        *consumed = 1;
        return 0;
    }
    if (((head >> 6) & 1) != 0) return -1;
    int sw = (head >> 4) & 0x03;
    int otype = head & 0x0F;
    if (otype > IDMIX_OTYPE_INT64) return -1;
    int num_bytes = SW_BYTES[sw];
    if (len < offset + 1 + (size_t)num_bytes) return -1;
    uint64_t raw = 0;
    for (int i = 0; i < num_bytes; i++) raw |= (uint64_t)data[offset + 1 + i] << (8 * i);
    tv->otype = otype;
    tv->val = reconstruct_int(otype, sw, raw);
    *consumed = (size_t)(1 + num_bytes);
    return 0;
}

static int encode_binary(idmix_ctx_t* ctx, const idmix_value_t* typed, size_t count, int variant_id,
                         byte_buf_t* out) {
    byte_buf_t objects;
    if (byte_buf_init(&objects, count * 9) != 0) return -1;
    uint8_t obj[16];
    int obj_len;
    for (size_t i = 0; i < count; i++) {
        if (encode_object(&typed[i], obj, &obj_len) != 0) {
            byte_buf_free(&objects);
            return -1;
        }
        if (byte_buf_append(&objects, obj, (size_t)obj_len) != 0) {
            byte_buf_free(&objects);
            return -1;
        }
    }
    uint8_t mask = (uint8_t)((variant_id * 0x9D + 0x37) & 0xFF);
    for (size_t i = 0; i < objects.len; i++) objects.data[i] ^= mask;

    int header = (variant_id << ctx->variant_shift) | ((int)count << ctx->count_shift);
    if (byte_buf_init(out, 2 + objects.len) != 0) {
        byte_buf_free(&objects);
        return -1;
    }
    out->data[0] = (uint8_t)(header & 0xFF);
    out->data[1] = (uint8_t)((header >> 8) & 0xFF);
    out->len = 2;
    byte_buf_append(out, objects.data, objects.len);
    byte_buf_free(&objects);

    int xor_sum = 0;
    for (size_t i = 0; i < out->len; i++) xor_sum ^= out->data[i];
    header |= xor_sum & ctx->check_mask;
    out->data[0] = (uint8_t)(header & 0xFF);
    out->data[1] = (uint8_t)((header >> 8) & 0xFF);
    return 0;
}

static int decode_binary(idmix_ctx_t* ctx, const uint8_t* data, size_t data_len, idmix_value_t** values,
                         size_t* count) {
    if (data_len < 2) return -1;
    int header = data[0] | (data[1] << 8);
    int check = header & ctx->check_mask;
    int obj_count = (header & ctx->count_mask) >> ctx->count_shift;
    int variant_id = (header & ctx->variant_mask) >> ctx->variant_shift;
    if (variant_id >= ctx->max_variants || obj_count > ctx->max_objects) return -1;

    uint8_t* verify = (uint8_t*)malloc(data_len);
    if (!verify) return -1;
    memcpy(verify, data, data_len);
    verify[0] &= (uint8_t)~ctx->check_mask;
    int xor_sum = 0;
    for (size_t i = 0; i < data_len; i++) xor_sum ^= verify[i];
    free(verify);
    if ((xor_sum & ctx->check_mask) != check) return -1;

    uint8_t* objects = (uint8_t*)malloc(data_len - 2);
    if (!objects) return -1;
    memcpy(objects, data + 2, data_len - 2);
    uint8_t mask = (uint8_t)((variant_id * 0x9D + 0x37) & 0xFF);
    for (size_t i = 0; i < data_len - 2; i++) objects[i] ^= mask;

    idmix_value_t* result = (idmix_value_t*)calloc((size_t)obj_count, sizeof(idmix_value_t));
    if (!result) {
        free(objects);
        return -1;
    }
    size_t pos = 0;
    for (int i = 0; i < obj_count; i++) {
        size_t consumed = 0;
        if (decode_object(objects, data_len - 2, pos, &result[i], &consumed) != 0) {
            free(objects);
            free(result);
            return -1;
        }
        pos += consumed;
    }
    free(objects);
    if (pos != data_len - 2) {
        free(result);
        return -1;
    }
    *values = result;
    *count = (size_t)obj_count;
    return 0;
}

int idmix_encode(idmix_ctx_t* ctx, const idmix_value_t* values, size_t count, char** out) {
    if (!ctx || !values || count == 0 || !out) return -1;
    if ((int)count > ctx->max_objects) return -1;
    int variant_id = (int)(rand() % ctx->max_variants);
    byte_buf_t data;
    if (encode_binary(ctx, values, count, variant_id, &data) != 0) return -1;
    *out = radix_encode_bytes(ctx, data.data, data.len);
    byte_buf_free(&data);
    return *out ? 0 : -1;
}

int idmix_decode(idmix_ctx_t* ctx, const char* s, idmix_value_t** values, size_t* count) {
    if (!ctx || !s || !values || !count) return -1;
    uint8_t* data = NULL;
    size_t data_len = 0;
    if (radix_decode_bytes(ctx, s, &data, &data_len) != 0) return -1;
    int rc = decode_binary(ctx, data, data_len, values, count);
    free(data);
    return rc;
}
