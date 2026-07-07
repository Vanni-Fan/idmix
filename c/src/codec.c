#include "codec.h"

#include <stdlib.h>
#include <string.h>

struct idmix_codec {
    char* alphabet;
    int base;
    int from_custom[256];
};

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

static idmix_codec_t* codec_init(const char* alphabet) {
    if (!alphabet) alphabet = IDMIX_DEFAULT_ALPHABET;
    size_t len = strlen(alphabet);
    if (len < 2) return NULL;

    idmix_codec_t* codec = (idmix_codec_t*)calloc(1, sizeof(idmix_codec_t));
    if (!codec) return NULL;
    codec->alphabet = (char*)malloc(len + 1);
    if (!codec->alphabet) {
        idmix_codec_destroy(codec);
        return NULL;
    }
    memcpy(codec->alphabet, alphabet, len + 1);
    codec->base = (int)len;
    for (int i = 0; i < 256; i++) codec->from_custom[i] = -1;
    for (size_t i = 0; i < len; i++) {
        unsigned char c = (unsigned char)alphabet[i];
        if (codec->from_custom[c] >= 0) {
            idmix_codec_destroy(codec);
            return NULL;
        }
        codec->from_custom[c] = (int)i;
    }
    return codec;
}

idmix_codec_t* idmix_codec_create(const char* alphabet) { return codec_init(alphabet); }

void idmix_codec_destroy(idmix_codec_t* codec) {
    if (!codec) return;
    free(codec->alphabet);
    free(codec);
}

void idmix_codec_free_string(char* s) { free(s); }
void idmix_codec_free_bytes(uint8_t* data) { free(data); }

static char* radix_encode_bytes(idmix_codec_t* codec, const uint8_t* data, size_t data_len) {
    if (data_len == 0) {
        char* s = (char*)malloc(2);
        if (!s) return NULL;
        s[0] = codec->alphabet[0];
        s[1] = '\0';
        return s;
    }
    uint8_t* wrapped = (uint8_t*)malloc(2 + data_len);
    if (!wrapped) return NULL;
    wrapped[0] = (uint8_t)((data_len >> 8) & 0xFF);
    wrapped[1] = (uint8_t)(data_len & 0xFF);
    memcpy(wrapped + 2, data, data_len);

    bigint_t n, q, zero;
    bigint_from_bytes(&n, wrapped, 2 + data_len);
    free(wrapped);
    bigint_from_bytes(&zero, (const uint8_t*)"\0", 1);

    char rev[512];
    int rlen = 0;
    while (n.len > 1 || n.data[0] != 0) {
        int rem = 0;
        if (bigint_divmod_small(&n, codec->base, &rem, &q) != 0) {
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
        rev[rlen++] = codec->alphabet[rem];
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

static int radix_decode_bytes(idmix_codec_t* codec, const char* s, uint8_t** out, size_t* out_len) {
    if (!s || !s[0]) return -1;
    bigint_t n;
    bigint_from_bytes(&n, (const uint8_t*)"\0", 1);
    for (const char* p = s; *p; p++) {
        unsigned char c = (unsigned char)*p;
        if (codec->from_custom[c] < 0) {
            bigint_free(&n);
            return -1;
        }
        if (bigint_muladd_small(&n, codec->base, codec->from_custom[c]) != 0) {
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
            int payload_len = (buf[0] << 8) | buf[1];
            if ((int)raw_len == 2 + payload_len) {
                *out = (uint8_t*)malloc((size_t)payload_len);
                if (!*out) {
                    free(buf);
                    bigint_free(&n);
                    return -1;
                }
                memcpy(*out, buf + 2, (size_t)payload_len);
                *out_len = (size_t)payload_len;
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

int idmix_codec_encode_bytes(idmix_codec_t* codec, const uint8_t* data, size_t data_len, char** out) {
    if (!codec || !out) return -1;
    *out = radix_encode_bytes(codec, data, data_len);
    return *out ? 0 : -1;
}

int idmix_codec_decode_bytes(idmix_codec_t* codec, const char* s, uint8_t** out, size_t* out_len) {
    if (!codec || !s || !out || !out_len) return -1;
    return radix_decode_bytes(codec, s, out, out_len);
}

int idmix_encode_bytes(const uint8_t* data, size_t data_len, char** out) {
    idmix_codec_t* codec = idmix_codec_create(NULL);
    if (!codec) return -1;
    int rc = idmix_codec_encode_bytes(codec, data, data_len, out);
    idmix_codec_destroy(codec);
    return rc;
}

int idmix_decode_string(const char* s, uint8_t** out, size_t* out_len) {
    idmix_codec_t* codec = idmix_codec_create(NULL);
    if (!codec) return -1;
    int rc = idmix_codec_decode_bytes(codec, s, out, out_len);
    idmix_codec_destroy(codec);
    return rc;
}
